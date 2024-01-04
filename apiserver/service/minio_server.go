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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	gqlconfig "github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
	"github.com/kubeagi/arcadia/pkg/cache"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

type (
	minioAPI struct {
		conf   gqlconfig.ServerConfig
		client dynamic.Interface
		lru    cache.Cache
	}

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
)

const (
	bucketQuery     = "bucket"
	bucketPathQuery = "bucketPath"
	md5Query        = "md5"
	cachePrefix     = "totallines"

	maxCSVLines     = 100
	namespaceHeader = "namespace"
)

/*
GetSuccessChunks
There are three different scenarios:

1. If the file exists, the function will return done=true and will not provide an uploadid.
In this case, no further action is required for uploading because the file is already present.

2. If the file does not exist, the function will return done=false.
In this case, you need to request a new uploadid to initiate the upload process.

3. If the upload is in progress, the function will return done=false, uploadid (e.g., uploadid=xx) and chunks={partNumber, etag}.
In this case, you can utilize the provided uploadid to continue requesting the upload URL and proceed with the file upload process.
*/
func (m *minioAPI) GetSuccessChunks(ctx *gin.Context) {
	fildMD5 := ctx.Query(md5Query)
	if fildMD5 == "" {
		klog.Error("md5 is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "md5 is required",
		})
		return
	}

	fileName := ctx.Query("fileName")
	bucketName := ctx.GetHeader(namespaceHeader)
	bucketPath := ctx.Query(bucketPathQuery)
	etag := ctx.Query("etag")
	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)

	r := SuccessChunksResult{
		Done: false,
	}

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	// First check if the file already exists in minio
	anyObject, err := source.StatFile(ctx.Request.Context(), &v1alpha1.OSS{Bucket: bucketName, Object: objectName})
	if err == nil {
		objectInfo, ok := anyObject.(minio.ObjectInfo)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "can't get file information",
			})
			return
		}
		if objectInfo.ETag == etag {
			// The file already exists and has the same md5, it is the same file and does not need to be uploaded again.
			r.Done = true
			ctx.JSON(http.StatusOK, r)
			return
		}
	}

	if err != nil && err.Error() != datasource.ObjectNotExistMsg {
		// When checking the existence of a file in MinIO, besides the "file not found" error, there can be other errors as well.
		klog.Errorf("failed to check for the existence of a resource %s/%s. error %s", bucketName, objectName, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	// If the file does not exist, you can check if there are any relevant upload records locally.
	uploadID, _ := source.IncompleteUpload(ctx.Request.Context(), datasource.WithBucket(bucketName),
		datasource.WithBucketPath(bucketPath), datasource.WithFileName(fileName))
	if uploadID == "" {
		ctx.JSON(http.StatusOK, r)
		return
	}

	// Checking already uploaded chunks
	r.UploadID = uploadID
	r.Chunks = make([]Chunk, 0)
	result, err := source.CompletedChunks(ctx.Request.Context(), datasource.WithBucket(bucketName),
		datasource.WithBucketPath(bucketPath), datasource.WithFileName(fileName),
		datasource.WithUploadID(uploadID))
	if err != nil {
		klog.Errorf("ListObjectParts failed: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	for _, objectPart := range result.(minio.ListObjectPartsResult).ObjectParts {
		r.Chunks = append(r.Chunks, Chunk{
			PartNumber: objectPart.PartNumber,
			ETag:       objectPart.ETag,
			Size:       objectPart.Size,
		})
	}
	ctx.JSON(http.StatusOK, r)
}

func (m *minioAPI) NewMultipart(ctx *gin.Context) {
	var body NewMultipartBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body to json structure error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "failed to parse body",
		})
		return
	}

	if body.ChunkCount > common.MaxPartsCount || body.ChunkCount <= 0 {
		klog.Errorf("illegal chunkCount %d", body.ChunkCount)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("chunkCount must be greater than zero and less than or equal to %d.", common.MaxPartsCount),
		})
		return
	}

	if body.Size > common.MaxMultipartPutObjectSize {
		klog.Error("illegal file size")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("the file size must be greater than or equal to 0 and less than or equal to %d.", common.MaxMultipartPutObjectSize),
		})
		return
	}

	if body.MD5 == "" {
		klog.Error("md5 is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "md5 is required",
		})
		return
	}
	if body.FileName == "" {
		klog.Error("file name is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "file name is required",
		})
		return
	}

	body.Bucket = ctx.GetHeader(namespaceHeader)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	uploadID, err := source.NewMultipartIdentifier(ctx.Request.Context(), datasource.WithBucket(body.Bucket),
		datasource.WithBucketPath(body.BucketPath), datasource.WithFileName(body.FileName),
		datasource.WithAnnotations(map[string]string{
			"size":              fmt.Sprintf("%d", body.Size),
			"creationTimestamp": time.Now().Format(time.RFC3339),
		}))
	if err != nil {
		klog.Errorf("failed to generate uploadid error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "failed to generate uploadid",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"uploadID": uploadID,
	})
}

/*
GetMultipartUploadURL
The function will check if the provided partNumber has been uploaded completely.

1. If it is completed, it will set complete=true.
2. If it is not completed, it will set complete=false and return the upload URL.
*/
func (m *minioAPI) GetMultipartUploadURL(ctx *gin.Context) {
	var body GenChunkURLBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "failed to parse body",
		})
		return
	}

	if body.MD5 == "" {
		klog.Error("md5 is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "md5 is required",
		})
		return
	}
	if body.UploadID == "" {
		klog.Error("uploadID is required")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "uploadID is required",
		})
		return
	}

	// FIXME: why
	if body.Size > common.MinPartSize {
		klog.Errorf("minimum slice is %d, current is %d", common.MinPartSize, body.Size)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("minimum part size is %d, current is %d", common.MinPartSize, body.Size),
		})
		return
	}
	if body.FileName == "" {
		klog.Errorf("fileName is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "fileName is required",
		})
		return
	}

	body.Bucket = ctx.GetHeader(namespaceHeader)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	result, err := source.CompletedChunks(ctx.Request.Context(), datasource.WithBucket(body.Bucket),
		datasource.WithBucketPath(body.BucketPath), datasource.WithFileName(body.FileName),
		datasource.WithUploadID(body.UploadID))
	if err != nil {
		klog.Errorf("ListObjectParts failed: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	for _, objectPart := range result.(minio.ListObjectPartsResult).ObjectParts {
		if objectPart.PartNumber == body.PartNumber {
			ctx.JSON(http.StatusOK, GenChunkURLResult{
				Completed: true,
			})
			return
		}
	}

	url, err := source.GenMultipartSignedURL(ctx.Request.Context(),
		datasource.WithBucket(body.Bucket),
		datasource.WithBucketPath(body.BucketPath),
		datasource.WithUploadID(body.UploadID),
		datasource.WithPartNumber(fmt.Sprintf("%d", body.PartNumber)),
		datasource.WithFileName(body.FileName))
	if err != nil {
		klog.Errorf("genMultiPartSignedURL failed: %s", err)
		klog.Errorf("failed to get multipart signed url error %s, md5 %s", err, body.MD5)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get multipart signed url",
		})
		return
	}

	ctx.JSON(http.StatusOK, GenChunkURLResult{
		Completed: false,
		URL:       url,
	})
}

func (m *minioAPI) CompleteMultipart(ctx *gin.Context) {
	var body CompleteBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "need content-type application/json",
		})
		return
	}
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	body.Bucket = ctx.GetHeader(namespaceHeader)
	err = source.Complete(ctx.Request.Context(),
		datasource.WithBucket(body.Bucket),
		datasource.WithBucketPath(body.BucketPath),
		datasource.WithUploadID(body.UploadID),
		datasource.WithFileName(body.FileName))
	if err != nil {
		klog.Errorf("complete multipart error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, "success")
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
	source, err := common.SystemDatasourceOSS(context.TODO(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	body.Bucket = ctx.GetHeader(namespaceHeader)
	bucketPath := strings.TrimSuffix(body.BucketPath, "/")
	for _, f := range body.Files {
		go func(fn string) {
			if err := source.Remove(context.TODO(), &v1alpha1.OSS{Bucket: body.Bucket, Object: fmt.Sprintf("%s/%s", bucketPath, fn)}); err != nil {
				klog.Errorf("failed to delete file %s/%s from bucket %s, error %s", bucketPath, fn, body.Bucket, err)
			}
		}(f)
	}
	ctx.JSON(http.StatusOK, "success")
}

func (m *minioAPI) Abort(ctx *gin.Context) {
	var body CompleteBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "need content-type application/json",
		})
		return
	}

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	body.Bucket = ctx.GetHeader(namespaceHeader)
	if err := source.Abort(ctx.Request.Context(), datasource.WithBucket(body.Bucket), datasource.WithBucketPath(body.BucketPath),
		datasource.WithFileName(body.FileName), datasource.WithUploadID(body.UploadID)); err != nil {
		klog.Errorf("failed to stop file upload, error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, "success")
}

func (m *minioAPI) StatFile(ctx *gin.Context) {
	fileName := ctx.Query("fileName")
	bucket := ctx.GetHeader(namespaceHeader)
	bucketPath := ctx.Query(bucketPathQuery)

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	anyObject, err := source.StatFile(ctx.Request.Context(), &v1alpha1.OSS{
		Object: fmt.Sprintf("%s/%s", bucketPath, fileName),
		Bucket: bucket,
	})
	if err != nil {
		klog.Errorf("stat file %s/%s/%s error %s", bucket, bucketPath, fileName, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	info, ok := anyObject.(minio.ObjectInfo)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "can't get file information",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"size": info.Size,
	})
}

func (m *minioAPI) Download(ctx *gin.Context) {
	fromStr := ctx.Query("from")
	endStr := ctx.Query("end")
	var (
		from, end int64
	)
	_, err := fmt.Sscanf(fromStr, "%d", &from)
	if err != nil {
		klog.Errorf("from %s is illegal, set default 0", fromStr)
		from = 0
	}
	_, err = fmt.Sscanf(endStr, "%d", &end)
	if err != nil {
		klog.Errorf("from %s is illegal, set default 0", fromStr)
		end = 0
	}

	bucket := ctx.GetHeader(namespaceHeader)
	bucketPath := ctx.Query(bucketPathQuery)
	fileName := ctx.Query("fileName")

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)
	opt := minio.GetObjectOptions{}
	_ = opt.SetRange(from, end)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	info, err := source.Client.GetObject(ctx.Request.Context(), bucket, objectName, opt)
	if err != nil {
		klog.Errorf("failed to get object %s/%s range %d-%d error %s", bucket, objectName, from, end, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	_, _ = io.Copy(ctx.Writer, info)
}

func (m *minioAPI) ReadCSVLines(ctx *gin.Context) {
	var (
		page       int64
		lines      int64
		totalLines int64

		bucket, bucketPath string
		fileName           string
	)
	_, _ = fmt.Sscanf(ctx.Query("page"), "%d", &page)
	_, _ = fmt.Sscanf(ctx.Query("size"), "%d", &lines)
	if page <= 0 {
		klog.Errorf("the minimum page should be 1")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "the minimum page should be 1",
		})
		return
	}
	if lines <= 0 || lines > maxCSVLines {
		klog.Errorf("the number of lines read should be greater than zero and less than or equal to %d", maxCSVLines)
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"message": fmt.Sprintf("the number of lines read should be greater than zero and less than or equal to %d", maxCSVLines),
		})
		return
	}
	bucket = ctx.GetHeader(namespaceHeader)
	bucketPath = ctx.Query(bucketPathQuery)
	fileName = ctx.Query("fileName")

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), nil, m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	anyStatInfo, err := source.StatFile(ctx.Request.Context(), &v1alpha1.OSS{Bucket: bucket, Object: objectName})
	if err != nil {
		klog.Errorf("failed to stat filed %s/%s error %s", bucket, objectName, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
	}
	statInfo := anyStatInfo.(minio.ObjectInfo)

	// checks if the total number of lines in the file has been cached
	key := [4]string{cachePrefix, bucket, bucketPath, statInfo.ETag}
	if a, ok := m.lru.Get(key); ok {
		if v, ok1 := a.(int64); ok1 {
			klog.V(5).Infof("nice, key: %v match, the file has not changed, and the total number of lines in the file is %d", key, v)
			totalLines = v
		}
	}

	object, err := source.Client.GetObject(context.TODO(), bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		klog.Errorf("failed to get data, error is %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer object.Close()

	startLine := (page-1)*lines + 1
	result, err := common.ReadCSV(object, startLine, lines, totalLines)
	if err != nil && err != io.EOF {
		klog.Errorf("there is an error reading the csv file, the error is %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	// cache the total number of lines in the file
	_ = m.lru.Set(key, result.Total)
	klog.V(5).Infof("set the total number of lines in the file to %d, key %v", result.Total, key)
	ctx.JSON(http.StatusOK, result)
}

func RegisterMinIOAPI(group *gin.RouterGroup, conf gqlconfig.ServerConfig) {
	c, err := client.GetClient(nil)
	if err != nil {
		panic(err)
	}

	lru, _ := cache.NewLRU(20)
	api := minioAPI{conf: conf, client: c, lru: lru}

	{
		// model apis
		group.GET("/model/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetSuccessChunks)
		group.POST("/model/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.NewMultipart)
		group.POST("/model/files/chunk_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.GetMultipartUploadURL)
		group.PUT("/model/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "models"), api.CompleteMultipart)
		group.PUT("/model/files/chunks/abort", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "models"), api.Abort)
		group.DELETE("/model/files", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "delete", "models"), api.DeleteFiles)
		group.GET("/model/files/stat", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.StatFile)
		group.GET("/model/files/download", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "models"), api.Download)
	}

	{
		// versioneddataset apis
		group.GET("/versioneddataset/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.GetSuccessChunks)
		group.POST("/versioneddataset/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.NewMultipart)
		group.POST("/versioneddataset/files/chunk_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.GetMultipartUploadURL)
		group.PUT("/versioneddataset/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "versioneddatasets"), api.CompleteMultipart)
		group.PUT("/versioneddataset/files/chunks/abort", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "create", "versioneddatasets"), api.Abort)
		group.DELETE("/versioneddataset/files", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "delete", "versioneddatasets"), api.DeleteFiles)
		group.GET("/versioneddataset/files/stat", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.StatFile)
		group.GET("/versioneddataset/files/download", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.Download)
		group.GET("/versioneddataset/files/csv", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "versioneddatasets"), api.ReadCSVLines)
	}
}
