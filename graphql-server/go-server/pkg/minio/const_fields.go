package minio

const (
	// The creation time of the file in minio,
	// in this case the generation time of the uplaodid when it was sliced.
	CreationTimestamp = "creationTimestamp"

	// This field refers to the file type
	// In the data returned by minio, there is usually content-type in userMetadata,
	// but this is made explicit by the content-type when the file is uploaded,
	// and if there are other types, we should need to determine them by userTags.
	FileContentType = "content-type"
)
