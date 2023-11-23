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

package minio

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"k8s.io/klog/v2"

	models "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/minio/model"
)

const (
	bucketQuery     = "bucket"
	bucketPathQuery = "bucket_path"
	md5Query        = "md5"
)

type SuccessChunksResult struct {
	ResultCode int    `json:"resultCode"`
	Uploaded   string `json:"uploaded"`
	UploadID   string `json:"uploadID"`
	Chunks     string `json:"chunks"`
}

type NewMultipartResult struct {
	UploadID string `json:"uploadID"`
}

type MultipartUploadURLResult struct {
	URL string `json:"url"`
}

// completeMultipartUpload container for completing multipart upload.
type CompleteMultipartUpload struct {
	XMLName xml.Name             `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload" json:"-"`
	Parts   []minio.CompletePart `xml:"Part"`
}

// completedParts is a collection of parts sortable by their part numbers.
// used for sorting the uploaded parts before completing the multipart request.
type completedParts []minio.CompletePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

const (
	PresignedUploadPartURLExpireTime = time.Hour * 24 * 7
)

func GetSuccessChunks(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var res = -1
	var uploaded, uploadID, chunks string
	var partNumberMarker, maxParts int

	query := req.URL.Query()
	fileMD5 := query.Get(md5Query)
	if fileMD5 == "" {
		klog.Error("GetSuccessChunks failed: md5 is required")
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("md5 is required"))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	for {
		fileChunk, err := models.GetFileChunkByMD5(fileMD5)
		if err != nil {
			klog.Infof("GetFileChunkByMD5 failed: %s", err)
			break
		}
		uploaded = strconv.Itoa(fileChunk.IsUploaded)
		uploadID = fileChunk.UploadID

		bucketName := query.Get(bucketQuery)
		bucktPath := query.Get(bucketPathQuery)
		objectName := fmt.Sprintf("%s/%s", bucktPath, fileChunk.FileName)

		isExist, err := isObjectExist(ctx, bucketName, objectName)
		if err != nil {
			klog.Errorf("isObjectExist failed: %s", err)
			break
		}

		if isExist {
			uploaded = "1"
			if fileChunk.IsUploaded != models.FileUploaded {
				klog.Info("the file has been uploaded but not recorded")
				fileChunk.IsUploaded = 1
				if err = models.UpdateFileChunk(fileChunk); err != nil {
					klog.Errorf("UpdateFileChunk failed: %s", err)
				}
			}
			res = 0
			break
		} else {
			uploaded = "0"
			if fileChunk.IsUploaded == models.FileUploaded {
				klog.Info("the file has been recorded but not uploaded")
				fileChunk.IsUploaded = 0
				if err = models.UpdateFileChunk(fileChunk); err != nil {
					klog.Errorf("UpdateFileChunk failed: %s", err)
				}
			}
		}
		_, client, err := GetClients()
		if err != nil {
			klog.Errorf("getClients failed: %s", err)
			break
		}

		// TODO partNumberMarker, maxParts ?
		listObjectPartsResult, err := client.ListObjectParts(ctx, bucketName, objectName, uploadID, partNumberMarker, maxParts)
		if err != nil {
			klog.Errorf("ListObjectParts failed: %s", err)
			break
		}
		for _, objectPart := range listObjectPartsResult.ObjectParts {
			chunks += strconv.Itoa(objectPart.PartNumber) + "-" + objectPart.ETag + ","
		}
		// nolint
		break
	}
	result := SuccessChunksResult{
		ResultCode: res,
		Uploaded:   uploaded,
		UploadID:   uploadID,
		Chunks:     chunks,
	}
	message, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(message)
	if err != nil {
		klog.Errorf("w.Write failed: %s", err)
	}
}

func NewMultipart(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var uploadID string
	query := req.URL.Query()
	queryTotalChunkCounts := query.Get("totalChunkCounts")
	if queryTotalChunkCounts == "" {
		klog.Error("NewMultipart failed: totalChunkCounts is required")
		_, err := w.Write([]byte("totalChunkCounts is required"))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	totalChunkCounts, err := strconv.Atoi(queryTotalChunkCounts)
	if err != nil {
		klog.Errorf("NewMultipart failed: %s", err)
		_, err := w.Write([]byte("totalChunkCounts is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	if totalChunkCounts > models.MaxPartsCount || totalChunkCounts <= 0 {
		klog.Error("totalChunkCounts is illegal.")
		_, err := w.Write([]byte("totalChunkCounts is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	querySize := query.Get("size")
	if querySize == "" {
		klog.Error("size is illegal.")
		_, err := w.Write([]byte("size is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	fileSize, err := strconv.ParseInt(querySize, 10, 64)
	if err != nil {
		klog.Error("size is illegal.")
		_, err := w.Write([]byte("size is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	if fileSize > models.MaxMultipartPutObjectSize || fileSize < 0 {
		klog.Error("size is illegal.")
		_, err := w.Write([]byte("size is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	md5 := query.Get(md5Query)
	if md5 == "" {
		klog.Error("md5 is illegal.")
		_, err := w.Write([]byte("md5 is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	fileName := query.Get("fileName")
	if fileName == "" {
		klog.Error("fileName is illegal.")
		_, err := w.Write([]byte("fileName is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	bucket := query.Get(bucketQuery)
	bucktPath := query.Get(bucketPathQuery)

	uploadID, err = newMultiPartUpload(ctx, bucket, bucktPath, fileName)
	if err != nil {
		klog.Errorf("newMultiPartUpload failed: %s", err)
		_, err := w.Write([]byte("newMultiPartUpload failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	_, err = models.InsetFileChunk(&models.FileChunk{
		UploadID:    uploadID,
		Md5:         md5,
		Size:        fileSize,
		FileName:    fileName,
		TotalChunks: totalChunkCounts,
	})
	if err != nil {
		klog.Errorf("InsetFileChunk failed: %s", err)
		_, err := w.Write([]byte("InsetFileChunk failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	result := NewMultipartResult{
		UploadID: uploadID,
	}

	w.Header().Set("Content-Type", "application/json")
	message, _ := json.Marshal(result)
	_, err = w.Write(message)
	if err != nil {
		klog.Errorf("w.Write failed: %s", err)
	}
}

func GetMultipartUploadURL(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var url string
	query := req.URL.Query()
	md5 := query.Get(md5Query)
	if md5 == "" {
		klog.Error("md5 is required.")
		_, err := w.Write([]byte("md5 is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	uploadID := query.Get("uploadID")
	if uploadID == "" {
		klog.Error("uploadID is required.")
		_, err := w.Write([]byte("uploadID is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	queryChunkNumber := query.Get("chunkNumber")
	if queryChunkNumber == "" {
		klog.Error("chunkNumber is required.")
		_, err := w.Write([]byte("chunkNumber is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	partNumber, err := strconv.Atoi(queryChunkNumber)
	if err != nil {
		klog.Errorf("chunkNumber is illegal: %s", err)
		_, err := w.Write([]byte("chunkNumber is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	querySize := query.Get("size")
	if querySize == "" {
		klog.Error("size is required.")
		_, err := w.Write([]byte("size is required."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	size, err := strconv.ParseInt(querySize, 10, 64)
	if err != nil {
		klog.Errorf("size is illegal: %s", err)
		_, err := w.Write([]byte("size is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	if size > models.MinPartSize {
		klog.Error("size is illegal.")
		_, err := w.Write([]byte("size is illegal."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	fileChunk, err := models.GetFileChunkByMD5(md5)
	if err != nil {
		klog.Errorf("GetFileChunkByUMD5 failed: %s", err)
		_, err := w.Write([]byte("GetFileChunkByMD5 failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	bucket := query.Get(bucketQuery)
	bucketPath := query.Get(bucketPathQuery)
	url, err = genMultiPartSignedURL(ctx, bucket, bucketPath, uploadID, partNumber, fileChunk.FileName, size)
	if err != nil {
		klog.Errorf("genMultiPartSignedURL failed: %s", err)
		_, err := w.Write([]byte("genMultiPartSignedURL failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	result := MultipartUploadURLResult{
		URL: url,
	}
	w.Header().Set("Content-Type", "application/json")
	message, _ := json.Marshal(result)
	_, err = w.Write(message)
	if err != nil {
		klog.Errorf("w.Write failed: %s", err)
	}
}

func CompleteMultipart(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	err := req.ParseForm()
	if err != nil {
		klog.Errorf("req.ParseForm failed: %s", err)
	}
	md5 := req.Form.Get(md5Query)
	uploadID := req.Form.Get("uploadID")
	fileChunk, err := models.GetFileChunkByMD5(md5)
	if err != nil {
		klog.Errorf("GetFileChunkByMD5 failed: %s", err)
		_, err := w.Write([]byte("GetFileChunkByMD5 failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	bucket := req.Form.Get(bucketQuery)
	bucketPath := req.Form.Get(bucketPathQuery)
	_, err = completeMultiPartUpload(ctx, bucket, bucketPath, uploadID, fileChunk.FileName)
	if err != nil {
		klog.Errorf("completeMultiPartUpload failed: %s", err)
		_, err := w.Write([]byte("completeMultiPartUpload failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	fileChunk.IsUploaded = models.FileUploaded

	err = models.UpdateFileChunk(fileChunk)
	if err != nil {
		klog.Errorf("UpdateFileChunk failed: %s", err)
		_, err := w.Write([]byte("completeMultiPartUpload failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte("success"))
	if err != nil {
		klog.Errorf("w.Write failed: %s", err)
	}
}

func UpdateMultipart(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		klog.Errorf("req.ParseForm failed: %s", err)
	}
	md5 := req.Form.Get(md5Query)
	etag := req.Form.Get("etag")
	chunkNumber := req.Form.Get("chunkNumber")
	fileChunk, err := models.GetFileChunkByMD5(md5)
	if err != nil {
		log.Println("GetFileChunkByMD5 failed")
		_, err := w.Write([]byte("GetFileChunkByMD5failed."))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}

	fileChunk.CompletedParts += chunkNumber + "-" + strings.ReplaceAll(etag, "\"", "") + ","

	err = models.UpdateFileChunk(fileChunk)
	if err != nil {
		klog.Errorf("UpdateFileChunk failed: %s", err)
		_, err := w.Write([]byte("UpdateFileChunk failed"))
		if err != nil {
			klog.Errorf("w.Write failed: %s", err)
		}
		return
	}
	_, err = w.Write([]byte("success"))
	if err != nil {
		klog.Errorf("w.Write failed: %s", err)
	}
}

func isObjectExist(ctx context.Context, bucketName string, objectName string) (bool, error) {
	isExist := false
	// TODO doneCh?
	doneCh := make(chan struct{})
	defer close(doneCh)

	client, _, err := GetClients()
	if err != nil {
		klog.Errorf("getClients failed: %s", err)
		return isExist, err
	}

	objectCh := client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: objectName, Recursive: false})
	object, ok := <-objectCh
	if !ok || object.Err != nil {
		klog.Errorf("ListObjects failed: %s", object.Err)
		return isExist, object.Err
	}
	isExist = true
	return isExist, nil
}

func newMultiPartUpload(ctx context.Context, bucketName, bucktPath, fileName string) (string, error) {
	_, minioClient, err := GetClients()
	if err != nil {
		klog.Errorf("getClient failed: %s", err)
		return "", err
	}

	objectName := fmt.Sprintf("%s/%s", bucktPath, fileName)

	return minioClient.NewMultipartUpload(ctx, bucketName, objectName, minio.PutObjectOptions{})
}

func genMultiPartSignedURL(ctx context.Context, bucket, bucketPath string, uploadID string, partNumber int, fileName string, partSize int64) (string, error) {
	_, client, err := GetClients()
	if err != nil {
		klog.Errorf("getClient failed: %s", err)
		return "", err
	}

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)
	u, err := client.Presign(ctx, http.MethodPut, bucket, objectName, PresignedUploadPartURLExpireTime, url.Values{
		"uploadId":   []string{uploadID},
		"partNumber": []string{strconv.Itoa(partNumber)},
	})
	if err != nil {
		klog.Errorf("Presign failed: %s", err)
		return "", err
	}
	return u.String(), nil
}

func completeMultiPartUpload(ctx context.Context, bucket, bucketPath string, uploadID string, fileName string) (string, error) {
	var partNumberMarker, maxParts int
	_, core, err := GetClients()
	if err != nil {
		klog.Errorf("getClient failed: %s", err)
		return "", err
	}

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)

	// TODO ? partNumberMarker, maxParts
	listObjectPartsResult, err := core.ListObjectParts(ctx, bucket, objectName, uploadID, partNumberMarker, maxParts)
	if err != nil {
		klog.Errorf("ListObjectParts failed: %s", err)
		return "", err
	}
	var completeMultipartUpload CompleteMultipartUpload
	for _, objectPart := range listObjectPartsResult.ObjectParts {
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, minio.CompletePart{
			PartNumber: objectPart.PartNumber,
			ETag:       objectPart.ETag,
		})
	}
	sort.Sort(completedParts(completeMultipartUpload.Parts))

	uploadInfo, err := core.CompleteMultipartUpload(ctx, bucket, objectName, uploadID, completeMultipartUpload.Parts, minio.PutObjectOptions{})
	if err != nil {
		klog.Errorf("CompleteMultipartUpload failed: %s", err)
		return "", err
	}
	return uploadInfo.ETag, nil
}
