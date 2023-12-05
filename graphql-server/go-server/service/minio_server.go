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
package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/graphql-server/go-server/config"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/auth"
	minio1 "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/minio"
	models "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/minio/model"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/oidc"
)

type minioAPI struct {
	conf config.ServerConfig
}

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

func (m *minioAPI) GetSuccessChunks(ctx *gin.Context) {
	res := -1
	var (
		uploaded, uploadID, chunks, objectName string
		partNumberMarker, maxParts             int
		listObjectPartsResult                  minio.ListObjectPartsResult
		client                                 *minio.Core
		exists                                 bool
	)

	fildMD5 := ctx.Query(md5Query)
	if fildMD5 == "" {
		klog.Error("md5 is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "md5 is required",
		})
		return
	}
	bucketName := ctx.Query(bucketQuery)
	bucketPath := ctx.Query(bucketPathQuery)

	fileChunk, err := models.GetFileChunkByMD5(bucketName, bucketPath, fildMD5)
	if err != nil {
		klog.Errorf("failed to get file chunk error %s", err)
		goto done
	}

	uploaded = fmt.Sprintf("%d", fileChunk.IsUploaded)
	uploadID = fileChunk.UploadID
	objectName = fmt.Sprintf("%s/%s", bucketPath, fileChunk.FileName)

	exists, err = isObjectExist(ctx.Request.Context(), bucketName, objectName)
	if err != nil {
		klog.Errorf("failed to check for the existence of a resource %s/%s. error %s", bucketName, objectName, err)
		goto done
	}
	if exists {
		uploaded = "1"
		if fileChunk.IsUploaded != models.FileUploaded {
			klog.Infof("the file %s/%s has been uploaded but not recorded.", bucketName, objectName)
			fileChunk.IsUploaded = 1
			if err = models.UpdateFileChunk(fileChunk); err != nil {
				klog.Errorf("failed to update file %s/%s upload status. error %s", bucketName, objectName, err)
			}
		}
		res = 0
		goto done
	} else {
		uploaded = "0"
		if fileChunk.IsUploaded == models.FileUploaded {
			klog.Infof("the file %s/%s has been recorded but not uploaded", bucketName, objectName)
			fileChunk.IsUploaded = 0
			if err = models.UpdateFileChunk(fileChunk); err != nil {
				klog.Errorf("failed to update file %s/%s upload status. error %s", bucketName, objectName, err)
			}
		}
	}

	_, client, err = minio1.GetClients()
	if err != nil {
		klog.Errorf("failed to get oss client. %s", err)
		goto done
	}

	listObjectPartsResult, err = client.ListObjectParts(ctx, bucketName, objectName, uploadID, partNumberMarker, maxParts)
	if err != nil {
		klog.Errorf("ListObjectParts failed: %s", err)
		goto done
	}
	for _, objectPart := range listObjectPartsResult.ObjectParts {
		chunks += fmt.Sprintf("%d-%s,", objectPart.PartNumber, objectPart.ETag)
	}

done:
	result := SuccessChunksResult{
		ResultCode: res,
		Uploaded:   uploaded,
		UploadID:   uploadID,
		Chunks:     chunks,
	}
	ctx.JSON(http.StatusOK, result)
}

func (m *minioAPI) NewMultipart(ctx *gin.Context) {
	var (
		uploadID    string
		totalChunks int
		size        uint64
	)
	totalChunksStr := ctx.Query("totalChunkCounts")
	_, err := fmt.Sscanf(totalChunksStr, "%d", &totalChunks)
	if err != nil {
		klog.Errorf("failed to get totalChunks error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	if totalChunks > models.MaxPartsCount || totalChunks <= 0 {
		klog.Errorf("illegal totalChunks %d", totalChunks)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("totalChunks must be greater than zero and less than or equal to %d.", models.MaxPartsCount),
		})
		return
	}

	fileSizeStr := ctx.Query("size")
	_, err = fmt.Sscanf(fileSizeStr, "%d", &size)
	if err != nil {
		klog.Errorf("failed to get file size error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	if size > models.MaxMultipartPutObjectSize {
		klog.Error("illegal file size")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("the file size must be greater than or equal to 0 and less than or equal to %d.", models.MaxMultipartPutObjectSize),
		})
		return
	}

	md5 := ctx.Query(md5Query)
	if md5 == "" {
		klog.Error("md5 is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "md5 is required",
		})
		return
	}
	fileName := ctx.Query("fileName")
	if fileName == "" {
		klog.Error("file name is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "file name is required",
		})
		return
	}

	bucket := ctx.Query(bucketQuery)
	bucketPath := ctx.Query(bucketPathQuery)
	fileType := ctx.Query("fileType")
	uploadID, err = newMultiPartUpload(ctx.Request.Context(), bucket, bucketPath, fileName, fileType, size)
	if err != nil {
		klog.Errorf("failed to generate uploadid error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "fialed to generate uploadid",
		})
		return
	}
	if _, err = models.InsetFileChunk(&models.FileChunk{
		UploadID:    uploadID,
		Md5:         md5,
		Size:        int64(size), // maybe need to change to uint64
		FileName:    fileName,
		TotalChunks: totalChunks,
		Bucket:      bucket,
		BucketPath:  bucketPath,
	}); err != nil {
		klog.Errorf("failed to insert new file chunk error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "failed to store new file chunk",
		})
		return
	}

	result := NewMultipartResult{
		UploadID: uploadID,
	}
	ctx.JSON(http.StatusOK, result)
}

func (m *minioAPI) GetMultipartUploadURL(ctx *gin.Context) {
	var (
		url   string
		parts int
		size  int64
	)
	md5 := ctx.Query(md5Query)
	if md5 == "" {
		klog.Error("md5 is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "md5 is required",
		})
		return
	}
	uploadID := ctx.Query("uploadID")
	if uploadID == "" {
		klog.Error("uploadID is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "uploadID is required",
		})
		return
	}

	_, err := fmt.Sscanf(ctx.Query("chunkNumber"), "%d", &parts)
	if err != nil {
		klog.Errorf("failed to get chunkNumber error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "failed to get chunkNumber, failed to get chunkNumber, please fill in the correct number.",
		})
		return
	}
	_, err = fmt.Sscanf(ctx.Query("size"), "%d", &size)
	if err != nil {
		klog.Errorf("failed to get file size error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "failed to get file size, please provide the correct file size.",
		})
		return
	}
	if size > models.MinPartSize {
		klog.Errorf("minimum slice is %d, current is %d", models.MinPartSize, size)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("minimum part size is %d, current is %d", models.MinPartSize, size),
		})
		return
	}

	bucket := ctx.Query(bucketQuery)
	bucketPath := ctx.Query(bucketPathQuery)
	fileChunk, err := models.GetFileChunkByMD5(bucket, bucketPath, md5)
	if err != nil {
		klog.Errorf("failed to get file chunk by md5 %s, error %s", md5, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get file chunk by md5",
		})
		return
	}

	url, err = genMultiPartSignedURL(ctx, bucket, bucketPath, uploadID, parts, fileChunk.FileName, size)
	if err != nil {
		klog.Errorf("genMultiPartSignedURL failed: %s", err)
		klog.Errorf("failed to get multipart signed url error %s, md5 %s", err, md5)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get multipart signed url",
		})
		return
	}

	result := MultipartUploadURLResult{
		URL: url,
	}
	ctx.JSON(http.StatusOK, result)
}

type CompleteBody struct {
	MD5        string `json:"md5"`
	BucketPath string `json:"bucket_path"`
	Bucket     string `json:"bucket"`
	FileName   string `json:"file_name"`
	Size       uint64 `json:"size"`
	UploadID   string `json:"uploadID"`
}

// CompleteMultipart why use form-data, compatible with front-end
func (m *minioAPI) CompleteMultipart(ctx *gin.Context) {
	var body CompleteBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "need content-type application/json",
		})
		return
	}
	fileChunk, err := models.GetFileChunkByMD5(body.Bucket, body.BucketPath, body.MD5)
	if err != nil {
		klog.Errorf("failed to get file chunk error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	_, err = completeMultiPartUpload(ctx.Request.Context(), body.Bucket, body.BucketPath, body.UploadID, fileChunk.FileName)
	if err != nil {
		klog.Errorf("complte multipart error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	fileChunk.IsUploaded = models.FileNotUploaded
	if err = models.UpdateFileChunk(fileChunk); err != nil {
		klog.Errorf("update file chunk error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, "success")
}

func (m *minioAPI) UpdateMultipart(ctx *gin.Context) {
	md5 := ctx.PostForm(md5Query)
	etag := ctx.PostForm("etag")
	chunkNumber := ctx.PostForm("chunkNumber")
	bucket := ctx.PostForm(bucketQuery)
	bucketPath := ctx.PostForm(bucketPathQuery)
	fileChunk, err := models.GetFileChunkByMD5(bucket, bucketPath, md5)
	if err != nil {
		klog.Errorf("failed to get file chunk error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get file chunk",
		})
		return
	}
	fileChunk.CompletedParts += fmt.Sprintf("%s-%s", chunkNumber, strings.ReplaceAll(etag, "\"", ""))
	err = models.UpdateFileChunk(fileChunk)
	if err != nil {
		klog.Errorf("failed to update file chunk error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, "success")
}

type DelteFileBody struct {
	Files      []string `json:"files"`
	Bucket     string   `json:"bucket"`
	BucketPath string   `json:"bucket_path"`
}

func (m *minioAPI) DeleteFiles(ctx *gin.Context) {
	var body DelteFileBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("bind json error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "can't bind to json",
		})
		return
	}
	client, _, err := minio1.GetClients()
	if err != nil {
		klog.Errorf("can't get oss client error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "can't get oss client",
		})
		return
	}
	bucketPath := strings.TrimSuffix(body.BucketPath, "/")
	for _, f := range body.Files {
		go func(fn string) {
			if err := client.RemoveObject(context.Background(), body.Bucket, fmt.Sprintf("%s/%s", bucketPath, fn), minio.RemoveObjectOptions{
				ForceDelete: true,
			}); err != nil {
				klog.Errorf("faile to delete file %s/%s from bucket %s, error %s", bucketPath, fn, body.Bucket, err)
			}
		}(f)
	}
	ctx.JSON(http.StatusOK, "success")
}

func (m *minioAPI) GetFile(ctx *gin.Context) {
	bucket := ctx.Query(bucketQuery)
	fileName := ctx.Query("fileName")
	client, _, err := minio1.GetClients()
	if err != nil {
		klog.Errorf("failed to get oss client error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "can't get oss client",
		})
		return
	}

	minioObject, err := client.GetObject(ctx.Request.Context(), bucket, fileName, minio.GetObjectOptions{})
	if err != nil {
		klog.Errorf("read file from %s/%s error %s", bucket, fileName, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	defer minioObject.Close()
	_, _ = io.Copy(ctx.Writer, minioObject)
}

func (m *minioAPI) UpdateFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		klog.Errorf("failed to read file content error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "please provide the correct form data",
		})
		return
	}
	bucket := ctx.PostForm(bucketQuery)
	if bucket == "" {
		klog.Warningf("bucket is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "bucket is required",
		})
		return
	}
	fileName := ctx.PostForm("fileName")
	if fileName == "" {
		klog.Warningf("fileName is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "fileName is required",
		})
		return
	}
	sizeStr := ctx.PostForm("size")
	if sizeStr == "" {
		klog.Warningf("size is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "size is required",
		})
		return
	}
	var size int64
	fmt.Sscanf(sizeStr, "%d", &size)
	client, _, err := minio1.GetClients()
	if err != nil {
		klog.Errorf("try to update file content, failed to get oss client error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	f, err := file.Open()
	if err != nil {
		klog.Errorf("can't open uploaded file")
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer f.Close()
	_, err = client.PutObject(ctx.Request.Context(), bucket, fileName, f, size, minio.PutObjectOptions{})
	if err != nil {
		klog.Errorf("failed to put object error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, "success")
}

func isObjectExist(ctx context.Context, bucketName string, objectName string) (bool, error) {
	isExist := false
	// TODO doneCh?
	doneCh := make(chan struct{})
	defer close(doneCh)

	client, _, err := minio1.GetClients()
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

func newMultiPartUpload(ctx context.Context, bucketName, bucktPath, fileName, fileType string, size uint64) (string, error) {
	_, minioClient, err := minio1.GetClients()
	if err != nil {
		klog.Errorf("getClient failed: %s", err)
		return "", err
	}

	objectName := fmt.Sprintf("%s/%s", bucktPath, fileName)

	return minioClient.NewMultipartUpload(ctx, bucketName, objectName, minio.PutObjectOptions{
		UserTags: map[string]string{
			"creationTimestamp": time.Now().Format(time.RFC3339),
			"size":              fmt.Sprintf("%d", size),
			"object_type":       fileType,
		},
	})
}

func genMultiPartSignedURL(ctx context.Context, bucket, bucketPath string, uploadID string, partNumber int, fileName string, partSize int64) (string, error) {
	_, client, err := minio1.GetClients()
	if err != nil {
		klog.Errorf("getClient failed: %s", err)
		return "", err
	}

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)
	u, err := client.Presign(ctx, http.MethodPut, bucket, objectName, PresignedUploadPartURLExpireTime, url.Values{
		"uploadId":   []string{uploadID},
		"partNumber": []string{fmt.Sprintf("%d", partNumber)},
	})
	if err != nil {
		klog.Errorf("Presign failed: %s", err)
		return "", err
	}
	return u.String(), nil
}

func completeMultiPartUpload(ctx context.Context, bucket, bucketPath string, uploadID string, fileName string) (string, error) {
	var partNumberMarker, maxParts int
	_, core, err := minio1.GetClients()
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

func RegisterMinIOAPI(e *gin.Engine, conf config.ServerConfig) {
	api := minioAPI{conf: conf}
	group := e.Group("/minio")

	group.GET("/model/files/get_chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetSuccessChunks)
	group.GET("/versioneddataset/files/get_chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.GetSuccessChunks)
	group.GET("get_chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetSuccessChunks)

	// POST
	group.GET("/model/files/new_multipart", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.NewMultipart)
	group.GET("/versioneddataset/files/new_multipart", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.NewMultipart)
	group.GET("new_multipart", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.NewMultipart)

	group.GET("/model/files/get_multipart_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetMultipartUploadURL)
	group.GET("/versioneddataset/files/get_multipart_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.GetMultipartUploadURL)
	group.GET("get_multipart_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetMultipartUploadURL)

	group.POST("/model/files/update_chunk", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "update", "models"), api.UpdateMultipart)
	group.POST("/versioneddataset/files/update_chunk", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "update", "versioneddatasets"), api.UpdateMultipart)
	group.POST("update_chunk", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "update", "models"), api.UpdateMultipart)

	group.POST("/model/files/complete_multipart", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "models"), api.CompleteMultipart)
	group.POST("/versioneddataset/files/complete_multipart", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "versioneddatasets"), api.CompleteMultipart)
	group.POST("complete_multipart", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "models"), api.CompleteMultipart)

	group.DELETE("/model/files/delete_files", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "delete", "models"), api.DeleteFiles)
	group.DELETE("/versioneddataset/files/delete_files", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "delete", "versioneddatasets"), api.DeleteFiles)

	group.GET("/versioneddataset/files/file", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.GetFile)
	group.POST("/versioneddataset/files/file", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "update", "versioneddatasets"), api.UpdateFile)

	group.GET("/model/files/file", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetFile)
	group.POST("/model/files/file", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "update", "models"), api.UpdateFile)
}
