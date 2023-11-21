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
	"errors"
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
	UUID           string
	Md5            string
	IsUploaded     int
	UploadID       string
	TotalChunks    int
	Size           int64
	FileName       string
	CompletedParts string
}

var fileChunks = make([]*FileChunk, 0, 250)

func GetFileChunkByMD5(md5 string) (*FileChunk, error) {
	fileChunk := new(FileChunk)
	matched := false
	for _, chunk := range fileChunks {
		if chunk.Md5 == md5 {
			fileChunk = chunk
			matched = true
		}
	}
	if !matched {
		return nil, errors.New("GetFileChunksByUUID failed")
	}
	return fileChunk, nil
}

func GetFileChunkByUUID(uuid string) (*FileChunk, error) {
	fileChunk := new(FileChunk)
	matched := false
	for _, chunk := range fileChunks {
		if chunk.UUID == uuid {
			fileChunk = chunk
			matched = true
			break
		}
	}
	if !matched {
		return nil, errors.New("GetFileChunksByUUID failed")
	}
	return fileChunk, nil
}

func UpdateFileChunk(fileChunk *FileChunk) error {
	updated := false
	for _, chunk := range fileChunks {
		if chunk.UUID == fileChunk.UUID && chunk.Md5 == fileChunk.Md5 {
			chunk.IsUploaded = fileChunk.IsUploaded
			chunk.CompletedParts = fileChunk.CompletedParts
			updated = true
		}
	}
	if !updated {
		return errors.New("UpdateFileChunk failed")
	}
	return nil
}

func InsetFileChunk(fileChunk *FileChunk) (_ *FileChunk, err error) {
	fileChunks = append(fileChunks, fileChunk)
	return fileChunk, nil
}
