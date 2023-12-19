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
package common

const (
	// The creation time of the file in minio,
	// in this case the generation time of the uplaodid when it was sliced.
	CreationTimestamp = "creationTimestamp"

	// This field refers to the file type
	// In the data returned by minio, there is usually content-type in userMetadata,
	// but this is made explicit by the content-type when the file is uploaded,
	// and if there are other types, we should need to determine them by userTags.
	FileContentType = "content-type"

	FileNotUploaded int = iota
	FileUploaded

	// maxPartsCount - maximum number of parts for a single multipart session.
	MaxPartsCount = 10000

	// maxMultipartPutObjectSize - maximum size 5TiB of object for
	// Multipart operation.
	MaxMultipartPutObjectSize = 1024 * 1024 * 1024 * 1024 * 5

	// minPartSize - minimum part size 128MiB per object after which
	// putObject behaves internally as multipart.
	MinPartSize = 1024 * 1024 * 64
)
