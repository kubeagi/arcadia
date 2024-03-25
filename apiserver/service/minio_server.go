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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	evaluationarcadiav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	gqlconfig "github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	pkgclient "github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	apiserverds "github.com/kubeagi/arcadia/apiserver/pkg/datasource"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
	"github.com/kubeagi/arcadia/apiserver/pkg/versioneddataset"
	"github.com/kubeagi/arcadia/pkg/cache"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

type (
	minioAPI struct {
		conf   gqlconfig.ServerConfig
		client client.Client
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

	WebCrawlerFileBody struct {
		VersionedDataset string `json:"versioneddataset" binding:"required"`
		Datasource       string `json:"datasource" binding:"required"`

		// Params to generate a web crawler file
		Params struct {
			URL *string `json:"url,omitempty"`
			// Params
			IntervalTime   *int     `json:"interval_time,omitempty"`
			ResourceTypes  []string `json:"resource_types,omitempty"`
			MaxDepth       int      `json:"max_depth,omitempty"`
			MaxCount       int      `json:"max_count,omitempty"`
			ExcludeSubUrls []string `json:"exclude_sub_urls,omitempty"`
			IncludeSubUrls []string `json:"include_sub_urls,omitempty"`
			ExcludeImgInfo struct {
				Weight int `json:"weight,omitempty"`
				Height int `json:"height,omitempty"`
			} `json:"exclude_img_info,omitempty"`
		} `json:"params"`
	}
)

const (
	bucketQuery     = "bucket"
	bucketPathQuery = "bucketPath"
	objectVersionID = "version"
	md5Query        = "md5"
	cachePrefix     = "totallines"

	maxCSVLines     = 100
	namespaceHeader = "namespace"
)

// @Summary	Get success chunks of a file
// @Schemes
// @Description	Get success chunks of a file
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			md5			query		string	true	"MD5 value of the file"
// @Param			fileName	query		string	true	"Name of the file"
// @Param			namespace	header		string	true	"Name of the bucket"
// @Param			bucketPath	query		string	true	"Path of the bucket"
// @Param			etag		query		string	true	"ETag of the file"
// @Success		200			{object}	SuccessChunksResult
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/chunks [get]
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
	bucketName := NamespaceInHeader(ctx)
	bucketPath := ctx.Query(bucketPathQuery)
	etag := ctx.Query("etag")
	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)

	r := SuccessChunksResult{
		Done: false,
	}

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
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

// @Summary	create new multipart upload
// @Schemes
// @Description	create new multipart upload
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		NewMultipartBody	true	"query params"
// @Param			namespace	header		string				true	"Name of the bucket"
// @Success		200			{object}	map[string]string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/chunks [post]
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

	body.Bucket = NamespaceInHeader(ctx)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
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

// @Summary	Get multipart upload URL
// @Schemes
// @Description	Get multipart upload URL
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		GenChunkURLBody	true	"query params"
// @Param			namespace	header		string			true	"Name of the bucket"
// @Success		200			{object}	GenChunkURLResult
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/chunk_url [post]
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

	body.Bucket = NamespaceInHeader(ctx)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
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

// @Summary	Complete multipart upload
// @Schemes
// @Description	Complete multipart upload
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		CompleteBody	true	"query params"
// @Param			namespace	header		string			true	"Name of the bucket"
// @Success		200			{object}	string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/chunks [put]
func (m *minioAPI) CompleteMultipart(ctx *gin.Context) {
	var body CompleteBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "need content-type application/json",
		})
		return
	}
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	body.Bucket = NamespaceInHeader(ctx)
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

// @Summary	Delete files
// @Schemes
// @Description	Delete files
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		DelteFileBody	true	"query params"
// @Param			namespace	header		string			true	"Name of the bucket"
// @Success		200			{object}	string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files [delete]
func (m *minioAPI) DeleteFiles(ctx *gin.Context) {
	var body DelteFileBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("bind json error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "can't bind to json",
		})
		return
	}
	source, err := common.SystemDatasourceOSS(context.TODO(), m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	body.Bucket = NamespaceInHeader(ctx)
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

// @Summary	Abort a file upload
// @Schemes
// @Description	Abort a file upload
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		CompleteBody	true	"query params"
// @Param			namespace	header		string			true	"Name of the bucket"
// @Success		200			{object}	string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/chunks/abort [put]
func (m *minioAPI) Abort(ctx *gin.Context) {
	var body CompleteBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "need content-type application/json",
		})
		return
	}

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	body.Bucket = NamespaceInHeader(ctx)
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

// @Summary	Statistics file information
// @Schemes
// @Description	Statistics file information
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			fileName	query		string	true	"Name of the file"
// @Param			namespace	header		string	true	"Name of the bucket"
// @Param			bucketPath	query		string	true	"Path of the bucket"
// @Param			version		query		string	false	"Download the specified version of the file, if not passed, download the latest version"
// @Success		200			{object}	map[string]string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/stat [get]
func (m *minioAPI) StatFile(ctx *gin.Context) {
	fileName := ctx.Query("fileName")
	bucket := NamespaceInHeader(ctx)
	bucketPath := ctx.Query(bucketPathQuery)

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	// versionID := ctx.Query(objectVersionID)
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

// @Summary	Download files in chunks
// @Schemes
// @Description	Download files in chunks
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			from		query	int		true	"The start of the file"
// @Param			end			query	int		true	"The end of the file"
// @Param			namespace	header	string	true	"Name of the bucket"
// @Param			bucketPath	query	string	true	"Path of the bucket"
// @Param			fileName	query	string	true	"Name of the file"
// @Param			version		query	string	false	"Download the specified version of the file, if not passed, download the latest version"
// @Success		200
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Router			/bff/model/files/download [get]
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

	bucket := NamespaceInHeader(ctx)
	bucketPath := ctx.Query(bucketPathQuery)
	fileName := ctx.Query("fileName")
	versionID := ctx.Query(objectVersionID)

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)
	opt := minio.GetObjectOptions{VersionID: versionID}
	_ = opt.SetRange(from, end)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
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

// @Summary	Read a file line by line
// @Schemes
// @Description	Read a file line by line
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			page		query		int		true	"Start page"
// @Param			size		query		int		true	"The number of rows read each time"
// @Param			namespace	header		string	true	"Name of the bucket"
// @Param			bucketPath	query		string	true	"Path of the bucket"
// @Param			fileName	query		string	true	"Name of the file"
// @Success		200			{object}	common.ReadCSVResult
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/versioneddataset/files/csv [get]
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
	bucket = NamespaceInHeader(ctx)
	bucketPath = ctx.Query(bucketPathQuery)
	fileName = ctx.Query("fileName")

	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
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
	key := [...]string{cachePrefix, bucket, bucketPath, statInfo.ETag, statInfo.VersionID}
	if a, ok := m.lru.Get(key); ok {
		if v, ok1 := a.(int64); ok1 {
			klog.V(5).Infof("nice, key: %v match, the file has not changed, and the total number of lines in the file is %d", key, v)
			totalLines = v
		}
	}

	object, err := source.Client.GetObject(context.TODO(), bucket, objectName, minio.GetObjectOptions{VersionID: statInfo.VersionID})
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
	result.Version = statInfo.VersionID
	ctx.JSON(http.StatusOK, result)
}

// @Summary	Get a download link
// @Schemes
// @Description	Get a download link
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			namespace	header		string	true	"Name of the bucket"
// @Param			bucketPath	query		string	true	"Path of the bucket"
// @Param			fileName	query		string	true	"Name of the file"
// @Param			version		query		string	false	"file version"
// @Success		200			{object}	map[string]string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/model/files/downloadlink [get]
func (m *minioAPI) GetDownloadLink(ctx *gin.Context) {
	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
	if err != nil {
		klog.Errorf("failed to get datasource %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	bucket := NamespaceInHeader(ctx)
	bucketPath := ctx.Query(bucketPathQuery)
	fileName := ctx.Query("fileName")
	objectName := fmt.Sprintf("%s/%s", bucketPath, fileName)

	v := url.Values{}
	if r := ctx.Query(objectVersionID); r != "" {
		v.Add("versionId", r)
	}
	u, err := source.Core.PresignedGetObject(ctx.Request.Context(), bucket, objectName, time.Hour*12, v)
	if err != nil {
		klog.Errorf("failed to generate download link %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"url": u.String()})
}

// @Summary	Create web cralwer file
// @Schemes
// @Description	Create a web crawler file which contains crawer params
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		WebCrawlerFileBody	true	"request params"
// @Param			namespace	header		string				true	"Name of the bucket"
// @Success		200			{object}	string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/versioneddataset/files/webcrawler [post]
func (m *minioAPI) CreateWebCrawlerFile(ctx *gin.Context) {
	var body WebCrawlerFileBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	namespace := NamespaceInHeader(ctx)

	// read versioneddataset
	vds, err := versioneddataset.GetVersionedDataset(ctx, m.client, body.VersionedDataset, namespace)
	if err != nil {
		klog.Errorf("failed to get versioneddataset error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to get versioneddataset error %s", err.Error()),
		})
		return
	}
	// read datasource
	ds, err := apiserverds.ReadDatasource(ctx, m.client, body.Datasource, namespace)
	if err != nil {
		klog.Errorf("failed to get datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to get datasource error %s", err.Error()),
		})
		return
	}
	if ds.Type != string(v1alpha1.DatasourceTypeWeb) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "not a web datasource",
		})
		return
	}

	if body.Params.URL == nil {
		body.Params.URL = ds.Endpoint.URL
	}
	if body.Params.IntervalTime == nil {
		body.Params.IntervalTime = ds.Web.RecommendIntervalTime
	}
	content, err := json.Marshal(body.Params)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("invalid web crawler params: %s", err.Error()),
		})
		return
	}

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to get system datasource error %s", err.Error()),
		})
		return
	}

	object := fmt.Sprintf("dataset/%s/%s/%s-%s.web", vds.Dataset.Name, vds.Version, ds.Namespace, ds.Name)
	_, err = source.Client.PutObject(ctx, namespace, object, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{})
	if err != nil {
		klog.Errorf("failed to put webcrawler file error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to put webcrawler file error %s", err.Error()),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"bucket": namespace, "object": object})
}

// @Summary	edit csv file online
// @Schemes
// @Description	edit csv file online
// @Tags			MinioAPI
// @Accept			json
// @Produce		json
// @Param			request		body		common.UpdateCSVBody	true	"request params"
// @Param			namespace	header		string					true	"Name of the bucket"
// @Success		200			{object}	string
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/bff/versioneddataset/files/edit [PUT]
func (m *minioAPI) EditCSV(ctx *gin.Context) {
	var body common.UpdateCSVBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		klog.Errorf("failed to parse body error %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	bucketName := NamespaceInHeader(ctx)
	if bucketName == "" {
		klog.Errorf("the namespace request header must be set")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "the namespace request header must be set",
		})
		return
	}

	objectName := fmt.Sprintf("%s/%s", body.BucketPath, body.FileName)
	if objectName == "" {
		klog.Errorf("unable to find target file, objectname is empty")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "unable to find target file",
		})
		return
	}

	source, err := common.SystemDatasourceOSS(ctx.Request.Context(), m.client)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to get system datasource error %s", err.Error()),
		})
		return
	}

	invalidVersion := body.Version == "" || body.Version == "null"
	latestVersion := ""
	singleFileVersions := source.Client.ListObjects(ctx.Request.Context(), bucketName, minio.ListObjectsOptions{WithVersions: true, Prefix: objectName})
	for fv := range singleFileVersions {
		if fv.IsLatest {
			latestVersion = fv.VersionID
			break
		}
	}
	if !invalidVersion && body.Version != latestVersion {
		// TODO: handle forceUpdate logic
		klog.Warningf("The latest version is %s, and version %s is ready to be edited. Please refresh the page and edit again.", latestVersion, body.Version)
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"message": fmt.Sprintf("The latest version is %s, and version %s is ready to be edited. Please refresh the page and edit again.", latestVersion, body.Version),
		})
		return
	}

	if !invalidVersion {
		swapFileName := fmt.Sprintf("/tmp/.%s", body.Version)
		sf, err := os.OpenFile(swapFileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
		if err != nil {
			if !os.IsExist(err) {
				klog.Errorf("an error occurred while creating, error %s", err)
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": "unable to lock file",
				})
				return
			}
			klog.Warningf("there are other users editing the collection, please try again later.")
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"message": "there are other users editing the collection, please try again later.",
			})
			return
		}
		sf.Close()
		defer os.Remove(swapFileName)
	}

	obj, err := source.Client.GetObject(ctx.Request.Context(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		klog.Errorf("failed to get original file error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	tags, _ := source.Client.GetObjectTagging(ctx.Request.Context(), bucketName, objectName, minio.GetObjectTaggingOptions{})
	buffer, l, err := common.EditCSV(obj, body.UpdateLines, body.NewLines, body.DelLines)
	if err != nil {
		klog.Errorf("generate new csv file failed error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	_, err = source.Client.PutObject(ctx.Request.Context(), bucketName, objectName, buffer, l, minio.PutObjectOptions{})
	if err != nil {
		klog.Errorf("write new csv file failed error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	_ = source.Client.PutObjectTagging(ctx.Request.Context(), bucketName, objectName, tags, minio.PutObjectTaggingOptions{})
	changed := len(body.DelLines) > 0 || len(body.UpdateLines) > 0 || len(body.NewLines) > 0

	if body.Knowledgebase != "" && changed {
		go func() {
			kb := v1alpha1.KnowledgeBase{}
			if err := m.client.Get(context.TODO(), types.NamespacedName{
				Namespace: bucketName,
				Name:      body.Knowledgebase,
			}, &kb); err != nil {
				klog.Errorf("failed to get knowledgebase %s, error %s", body.Knowledgebase, err)
				return
			}
			if kb.Annotations == nil {
				kb.Annotations = make(map[string]string)
			}
			now := time.Now().Format(time.RFC3339)
			kb.Annotations[v1alpha1.UpdateSourceFileAnnotationKey] = now
			if err := m.client.Update(context.TODO(), &kb); err != nil {
				klog.Errorf("failed to update knowledgebase %s, error %s", body.Knowledgebase, err)
				return
			}
			klog.Infof("successfully update knowledgebase %s", body.Knowledgebase)
		}()
	}
	ctx.JSON(http.StatusOK, "")
}

func registerMinIOAPI(group *gin.RouterGroup, conf gqlconfig.ServerConfig) {
	c, err := pkgclient.GetClient(nil)
	if err != nil {
		panic(err)
	}

	lru, _ := cache.NewLRU(20)
	api := minioAPI{conf: conf, client: c, lru: lru}

	{
		// model apis
		group.GET("/model/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "models"), api.GetSuccessChunks)
		group.POST("/model/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "models"), api.NewMultipart)
		group.POST("/model/files/chunk_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "models"), api.GetMultipartUploadURL)
		group.PUT("/model/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "create", "models"), api.CompleteMultipart)
		group.PUT("/model/files/chunks/abort", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "create", "models"), api.Abort)
		group.DELETE("/model/files", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "delete", "models"), api.DeleteFiles)
		group.GET("/model/files/stat", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "models"), api.StatFile)
		group.GET("/model/files/download", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "models"), api.Download)
		group.GET("/model/files/downloadlink", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "models"), api.GetDownloadLink)
	}

	{
		// versioneddataset apis
		group.GET("/versioneddataset/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.GetSuccessChunks)
		group.POST("/versioneddataset/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.NewMultipart)
		group.POST("/versioneddataset/files/chunk_url", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.GetMultipartUploadURL)
		group.PUT("/versioneddataset/files/chunks", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "create", "versioneddatasets"), api.CompleteMultipart)
		group.PUT("/versioneddataset/files/chunks/abort", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "create", "versioneddatasets"), api.Abort)
		group.DELETE("/versioneddataset/files", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "delete", "versioneddatasets"), api.DeleteFiles)
		group.GET("/versioneddataset/files/stat", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.StatFile)
		group.GET("/versioneddataset/files/download", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.Download)
		group.GET("/versioneddataset/files/csv", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.ReadCSVLines)
		group.GET("/versioneddataset/files/downloadlink", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.GetDownloadLink)
		// create a webcrawler file for versioneddataset
		group.POST("/versioneddataset/files/webcrawler", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "create", "versioneddatasets"), api.CreateWebCrawlerFile)
		group.PUT("/versioneddataset/files/edit", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "versioneddatasets"), api.EditCSV)
	}

	group.GET("/rags/files/downloadlink", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, evaluationarcadiav1alpha1.GroupVersion, "get", "rags"), api.GetDownloadLink)
}
