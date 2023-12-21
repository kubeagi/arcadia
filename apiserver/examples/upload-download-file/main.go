/*
Copyright 2023 KubeAGI.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

var (
	action     = flag.String("action", "upload", "you can only choose download, upload.")
	host       = flag.String("host", "http://localhost:8099", "apiserver address")
	fileName   = flag.String("file", "", "if it's an uploaded file, then it's the path to the local file, if it's a downloaded file, it's the path in minio, remember, bucketPath+filename make up the full storage path in minio.")
	bucket     = flag.String("bucket", "abc", "")
	bucketPath = flag.String("bucket-path", "dataset/ds1/v1", "")
	token      = flag.String("token", "", "bearer token")
)

const (
	bufSize = 1024 * 1024 * 32 // 32M

	chunksApi       = "/bff/versioneddataset/files/chunks"
	chunksURLApi    = "/bff/versioneddataset/files/chunk_url"
	statFileApi     = "/bff/versioneddataset/files/stat"
	downloadFileApi = "/bff/versioneddataset/files/download"

	upload   = "upload"
	download = "download"
)

type (
	Chunk struct {
		PartNumber int    `json:"partNubmer"`
		ETag       string `json:"etag"`
		Size       int64  `json:"size"`
	}
	SuccessChunksResult struct {
		Done     bool    `json:"done"`
		UploadID string  `json:"uploadID,omitempty"`
		Chunks   []Chunk `json:"chunks,omitempty"`
	}

	NewMultipartBody struct {
		// How many pieces do we have to divide it into?
		ChunkCount int `json:"chunkCount"`
		// part size
		Size int64 `json:"size"`
		// file md5
		MD5 string `json:"md5"`

		// The file is eventually stored in bucketPath/filtName in the bucket.
		Bucket     string `json:"bucket"`
		FileName   string `json:"fileName"`
		BucketPath string `json:"bucketPath"`
	}

	GenChunkURLBody struct {
		PartNumber int    `json:"partNumber"`
		Size       int64  `json:"size"`
		MD5        string `json:"md5"`
		UploadID   string `json:"uploadID"`
		Bucket     string `json:"bucket"`
		FileName   string `json:"fileName"`
		BucketPath string `json:"bucketPath"`
	}
	GenChunkURLResult struct {
		Completed bool   `json:"completed"`
		URL       string `json:"url"`
	}

	DelteFileBody struct {
		Files      []string `json:"files"`
		Bucket     string   `json:"bucket"`
		BucketPath string   `json:"bucketPath"`
	}

	CompleteBody struct {
		MD5        string `json:"md5"`
		UploadID   string `json:"uploadID"`
		Bucket     string `json:"bucket"`
		FileName   string `json:"fileName"`
		BucketPath string `json:"bucketPath"`
	}

	ReadCSVResp struct {
		Rows  [][]string `json:"rows"`
		Total int64      `json:"total"`
	}
)

func successChunks(md5, bucket, bucketPath, fileName, etag string, transport http.RoundTripper) (SuccessChunksResult, error) {
	klog.Infof("[DEBUG] check success chunks...")
	values := url.Values{}
	values.Add("md5", md5)
	values.Add("bucket", bucket)
	values.Add("bucketPath", bucketPath)
	values.Add("fileName", fileName)
	values.Add("etag", etag)
	api := *host + chunksApi + fmt.Sprintf("?%s", values.Encode())
	klog.Infof("[DEBUG] send get request to %s", api)

	req, _ := http.NewRequest(http.MethodGet, api, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	c := http.Client{Transport: transport}
	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("[Error] send request error %s", err)
		return SuccessChunksResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		klog.Infof("[Error] status code is %s, debug resp information %+v\n", resp.StatusCode, *resp)
		return SuccessChunksResult{}, fmt.Errorf("response code is %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return SuccessChunksResult{}, err
	}
	var result SuccessChunksResult
	if err := json.Unmarshal(data, &result); err != nil {
		klog.Errorf("[Error] can't unmarshal completes chunks error %s", err)
		return SuccessChunksResult{}, err
	}
	return result, nil
}

func newMultipart(
	md5, bucket, bucketPath, fileName string,
	partSize int64, chunkCount int,
	transport http.RoundTripper) (string, error) {

	klog.Infof("[DEBUG] request new multipart uploadid...")

	body := NewMultipartBody{
		ChunkCount: chunkCount,
		Size:       bufSize,
		MD5:        md5,
		FileName:   fileName,
		Bucket:     bucket,
		BucketPath: bucketPath,
	}
	bodyBytes, _ := json.Marshal(body)
	api := *host + chunksApi
	klog.Infof("[DEBUG] send post request to %s, with body %s...", api, string(bodyBytes))

	req, _ := http.NewRequest(http.MethodPost, api, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))

	req.Header.Set("Content-Type", "application/json")
	c := http.Client{Transport: transport}
	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("[Error] send newMultipart request error %s", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		klog.Infof("[Error] status code is %s, debug resp information %+v\n", resp.StatusCode, *resp)
		return "", fmt.Errorf("response code is %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("[Error] failed to read response body error %s", err)
		return "", err
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		klog.Errorf("[Error] can't unmarshal completes chunks error %s", err)
		return "", err
	}

	return result["uploadID"], nil
}

func genURL(
	md5, bucket, bucketPath, uploadID string,
	partNumer int, fileName string,
	transport http.RoundTripper) (GenChunkURLResult, error) {

	klog.Infof("[DEBUG] request upload url by uploadid: %s...", uploadID)
	body := GenChunkURLBody{
		PartNumber: partNumer,
		MD5:        md5,
		Bucket:     bucket,
		BucketPath: bucketPath,
		UploadID:   uploadID,
		Size:       bufSize,
		FileName:   fileName,
	}

	bodyBytes, _ := json.Marshal(body)
	api := *host + chunksURLApi
	klog.Infof("[DEBUG] send post request to %s, with body %s...", api, string(bodyBytes))

	req, _ := http.NewRequest(http.MethodPost, api, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))

	req.Header.Set("Content-Type", "application/json")
	c := http.Client{Transport: transport}
	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("[Error] send genMultipartURL request error %s", err)
		return GenChunkURLResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		klog.Infof("[Error] status code is %s, debug resp information %+v\n", resp.StatusCode, *resp)
		return GenChunkURLResult{}, fmt.Errorf("response code is %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("[Error] failed to read response body error %s", err)
		return GenChunkURLResult{}, err
	}
	var result GenChunkURLResult
	if err := json.Unmarshal(data, &result); err != nil {
		klog.Errorf("[Error] can't unmarshal completes chunks error %s", err)
		return GenChunkURLResult{}, err
	}
	return result, nil
}

func complete(md5, bucket, bucketPath, uploadID, fileName string, transport http.RoundTripper) error {
	klog.Infof("[DEBUG] all chunks are uploaded, merge all chunks...")
	body := CompleteBody{
		MD5:        md5,
		Bucket:     bucket,
		BucketPath: bucketPath,
		UploadID:   uploadID,
		FileName:   fileName,
	}

	bodyBytes, _ := json.Marshal(body)
	api := *host + chunksApi
	klog.Infof("[DEBUG] send put request to %s, with body %s...", api, string(bodyBytes))

	req, _ := http.NewRequest(http.MethodPut, api, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))

	req.Header.Set("Content-Type", "application/json")
	c := http.Client{Transport: transport}
	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("[Error] send complete chunks request error %s", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Infof("[Error] status code is %s, debug resp information %+v\n", resp.StatusCode, *resp)
		return fmt.Errorf("response code is %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	return nil
}

func fileMD5andEtag(f io.Reader) (string, string) {
	buf := make([]byte, bufSize)
	h := md5.New()
	etag := md5.New()
	parts := 0
	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return "", ""
		}
		if n == 0 {
			break
		}
		h.Write(buf[:n])
		tmp := md5.Sum(buf[:n])
		klog.Infof("[DEBUG] ***** part %d, md5 is %x\n", parts, tmp)
		etag.Write(tmp[:])
		parts++
	}
	return fmt.Sprintf("%x", h.Sum(nil)), fmt.Sprintf("%x-%d", etag.Sum(nil), parts)
}

func do(
	wg *sync.WaitGroup,
	f io.Reader,
	partNumber int, size int64,
	md5, uploadID, bucket, bucketPath, fileName string,
	transport http.RoundTripper) error {
	defer wg.Done()
	urlResult, err := genURL(md5, bucket, bucketPath, uploadID, partNumber, fileName, transport)
	if err != nil {
		klog.Error("[do Error] failed to gen url error %s", err)
		return err
	}
	if urlResult.Completed {
		klog.Infof("[do DEBUG] chunks have been uploaded successfully, skip")
		return nil
	}
	req, _ := http.NewRequest(http.MethodPut, urlResult.URL, f)
	c := http.Client{Transport: transport}
	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("[do Error] failed to upload file error ", err)
		return err
	}
	if resp.StatusCode/100 != 2 {
		klog.Errorf("[do Error] expect 200 get %d", resp.StatusCode)
		return fmt.Errorf("not 200")
	}
	return nil
}

func uploadFile(filePath, bucket, bucketPath string, tp http.RoundTripper) {
	f, err := os.Open(filePath)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	md5, etag := fileMD5andEtag(f)
	if err != nil {
		panic(err)
	}

	klog.Infof("[DEBUG] file md5 %s, etag: %s...", md5, etag)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		klog.Errorf("[Error] can't stat file size...")
		return
	}
	fileName := filePath
	if filepath.IsAbs(fileName) {
		fileName = strings.TrimPrefix(fileName, "/")
	}
	step := 1

	klog.Infof("[Step %d] check the number of chunks the file has completed.", step)
	step++
	completedChunks, err := successChunks(md5, bucket, bucketPath, fileName, etag, tp)
	if err != nil {
		klog.Errorf("[!!!Error] failed to check completed chunks. error %s", err)
		return
	}
	if completedChunks.Done {
		klog.Infof("[Done], the file already exists and does not need to be uploaded again")
		return
	}
	fileSize := fileInfo.Size()
	chunkCount := fileSize / bufSize
	if fileSize%bufSize != 0 {
		chunkCount++
	}
	if completedChunks.UploadID == "" {
		klog.Infof("[Step %d] get new uploadid", step)
		step++

		uploadID, err := newMultipart(md5, bucket, bucketPath, fileName, bufSize, int(chunkCount), tp)
		if err != nil {
			klog.Errorf("[!!!Error] failed to get new uplaodid. error %s", err)
			return
		}
		completedChunks.UploadID = uploadID
	}

	klog.Infof("[Step %d] tart uploading files based on uploadid %s.", step, completedChunks.UploadID)
	step++

	chunksMap := make(map[int]struct{})
	for _, chunk := range completedChunks.Chunks {
		klog.Infof("[DEBUG] complete partNubmer %d, etag %s, size: %d", chunk.PartNumber, chunk.ETag, chunk.Size)
		chunksMap[chunk.PartNumber] = struct{}{}
	}
	doComplete := true
	lock := make(chan struct{}, 1)
	var wg sync.WaitGroup
	for pn := 1; pn <= int(chunkCount); pn++ {
		if _, ok := chunksMap[pn]; ok {
			klog.Infof("[DEBUG] partNumber %d has already been uploaded, skip it.", pn)
			continue
		}
		wg.Add(1)

		reader := io.NewSectionReader(f, int64(pn-1)*bufSize, bufSize)
		go func(partNumber int, reader io.Reader) {
			if err := do(&wg, reader, partNumber, bufSize, md5, completedChunks.UploadID, bucket, bucketPath, fileName, tp); err != nil {
				klog.Errorf("!!![Error] Uploading the %d(st,ne,rd,th) chunk of the file, an error occurs, but the operation will not affect the other chunks at this time, so only the error will be logged here.", partNumber)
				lock <- struct{}{}
				doComplete = false
				<-lock
			}
		}(pn, reader)
	}
	wg.Wait()

	if doComplete {
		klog.Infof("[Step %d], all chunks are uploaded successfully and merging of chunks begins.", step)
		step++
		retryTimes := 1
		if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
			return true
		}, func() error {
			if err := complete(md5, bucket, bucketPath, completedChunks.UploadID, fileName, tp); err != nil {
				klog.Errorf("[!!!RetryError] retry %d, error %v", retryTimes, err)
				retryTimes++
			}
			return err
		}); err != nil {
			klog.Errorf("[!!!Error] After several retries, it still fails, please execute the subroutine again.")
			return
		}
	}

	klog.Infof("[Step %d], Congratulations, the file was uploaded successfully", step)
}

func downloadFile(bucket, bucketPath, fileName string, transport http.RoundTripper) {
	klog.Info("[Step 1] get file size")
	values := url.Values{}
	values.Add("fileName", fileName)
	values.Add("bucket", bucket)
	values.Add("bucketPath", bucketPath)

	c := http.Client{Transport: transport}
	api := fmt.Sprintf("%s%s?%s", *host, statFileApi, values.Encode())
	req, _ := http.NewRequest(http.MethodGet, api, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("[Error] failed to get file size")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		klog.Errorf("[DEBUG] expect 200 get statuscode %d, debug resp: %+v\n", resp.StatusCode, *resp)
		return
	}
	data, _ := io.ReadAll(resp.Body)
	var result map[string]int64
	if err := json.Unmarshal(data, &result); err != nil {
		klog.Errorf("[Error] failed to parse body error %s", err)
		return
	}
	size := result["size"]
	klog.Infof("[DEBUG] file size is %d", size)
	// TODO: Support for user-defined storage path later to do,
	// the current file downloads are stored directly in the tmp.gz this file,
	// download the completion of their own modification of the file suffix can be.
	klog.Infof("[Step 2] create local file tmp.gz")
	f, err := os.Create("tmp.gz")
	if err != nil {
		klog.Errorf("[Error] failed to create file with size %d", size)
		return
	}
	defer f.Close()
	if err = f.Truncate(size); err != nil {
		klog.Error("[Error] failed truncate to size %d", size)
		return
	}

	bufSize := int64(10 * 1024 * 1024)
	parts := size / int64(bufSize)
	if size%int64(bufSize) != 0 {
		parts++
	}
	from, end := int64(0), bufSize
	first := true
	lock := make(chan struct{}, 1)
	done := true
	klog.Infof("[Step 3] start to donwload...")
	var wg sync.WaitGroup
	for i := 0; i < int(parts); i++ {
		if !first {
			from, end = end, end+bufSize
		}
		first = false
		if end > size {
			end = size
		}

		wg.Add(1)
		go func(from, end int64) {
			defer wg.Done()
			v := url.Values{}
			v.Add("from", fmt.Sprintf("%d", from))
			v.Add("end", fmt.Sprintf("%d", end))
			v.Add("bucket", bucket)
			v.Add("bucketPath", bucketPath)
			v.Add("fileName", fileName)
			api := fmt.Sprintf("%s%s?%s", *host, downloadFileApi, v.Encode())
			klog.Infof("[Chunk %d-%d] send request to %s", from, end, api)

			req, _ := http.NewRequest(http.MethodGet, api, nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))

			resp, err := c.Do(req)
			if err != nil {
				klog.Errorf("failed download from %s to %d", from, end)
				lock <- struct{}{}
				done = false
				<-lock
				return
			}
			defer resp.Body.Close()
			lock <- struct{}{}
			f.Seek(from, io.SeekStart)
			io.Copy(f, resp.Body)
			<-lock
		}(from, end)
	}
	wg.Wait()
	if done {
		klog.Infof("[Step 4] File download complete")
	} else {
		klog.Errorf("[Error] Here there are some chunks that did not finish downloading, check for errors through the logs")
	}
}

func main() {
	flag.Parse()

	tp := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}

	if *action == upload {
		filepath.WalkDir(*fileName, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				klog.Errorf("[Error] failed access a path %s: %s", path, err)
				return err
			}
			if !d.IsDir() {
				uploadFile(path, *bucket, *bucketPath, tp)
			}
			return nil
		})
	} else {
		downloadFile(*bucket, *bucketPath, *fileName, tp)
	}
}
