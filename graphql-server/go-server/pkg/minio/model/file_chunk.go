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

package models

import (
	"fmt"
	"sync"
)

const (
	FileNotUploaded int = iota
	FileUploaded
)

// maxPartsCount - maximum number of parts for a single multipart session.
const MaxPartsCount = 10000

// maxMultipartPutObjectSize - maximum size 5TiB of object for
// Multipart operation.
const MaxMultipartPutObjectSize = 1024 * 1024 * 1024 * 1024 * 5

// minPartSize - minimum part size 128MiB per object after which
// putObject behaves internally as multipart.
const MinPartSize = 1024 * 1024 * 64

type FileChunk struct {
	Md5            string
	IsUploaded     int
	UploadID       string
	TotalChunks    int
	Size           int64
	FileName       string
	CompletedParts string
}

var fileChunks = sync.Map{}

func GetFileChunkByMD5(md5 string) (*FileChunk, error) {
	v, ok := fileChunks.Load(md5)
	if !ok {
		return nil, fmt.Errorf("not found chunk with md5 %s", md5)
	}
	vv, ok := v.(*FileChunk)
	if !ok {
		return nil, fmt.Errorf("error object")
	}
	return vv, nil
}

func UpdateFileChunk(fileChunk *FileChunk) error {
	v, err := GetFileChunkByMD5(fileChunk.Md5)
	if err != nil {
		return err
	}
	v.IsUploaded = fileChunk.IsUploaded
	v.CompletedParts = fileChunk.CompletedParts
	return nil
}

func InsetFileChunk(fileChunk *FileChunk) (_ *FileChunk, err error) {
	fileChunks.Store(fileChunk.Md5, fileChunk)
	return fileChunk, nil
}
