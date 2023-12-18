## Introduction

The back-end code implements chunked uploads as well as chunked downloads. Here is the complete set of calling logic.

### Build the code and get the executable

```shell
go build -o main main.go
```

### View command line arguments

```shell
Usage of ./main:
  -action string
    	you can only choose download, upload. (default "upload")
  -bucket string
    	(default "abc")
  -bucket-path string
    	(default "dataset/ds1/v1")
  -file string
    	if it's an uploaded file, then it's the path to the local file, if it's a downloaded file, it's the path in minio, remember, bucketPath+filename make up the full storage path in minio.
  -host string
    	apiserver address (default "http://localhost:8099")
```

- The `action` parameter specifies whether you want to upload or download a file.
- The `bucket` specifies the name of the bucket in the object store.
- The `bucket-path` indicates the path prefix of the file to be stored. For example, `bucket-path=abc/def`, and you want to store the file under `text/a.txt`, then the final storage path is abc/def/text/a.txt
- If you are uploading a file, then file is the storage path of the local file, **if you write absolute path /local/a.txt, bucket-path=abc/def then your file will be stored under abc/def/local/a.txt eventually**.
If you want to download a file, **specify the path after removing the bucket-path**, or the above example, you just need to write file=local/a.txt and it will download the file `abc/def/local/a.txt`.
- The host is the address of the backend service.

### Example of use


1. Upload local file, bucket is abc, bucket-path=def, local file is local/a.txt

```shell
./main --file=tmp.tar.gz --bucket=abc --bucket-path=def

I1211 15:15:56.834382 2903826 main.go:276] [DEBUG] ***** part 0, md5 is 8e3192f72fba1faee864edaa6f1636fc
I1211 15:15:56.889742 2903826 main.go:276] [DEBUG] ***** part 1, md5 is 29ee8a4e3f980fb9fe8b572c39d3caea
I1211 15:15:56.889786 2903826 main.go:323] [DEBUG] file md5 7ed8b2a8ea5798202b81eedf14dc978c, etag: 70b40381f8881546c265afe4f279cd01-2...
I1211 15:15:56.889808 2903826 main.go:331] [Step 1] check the number of chunks the file has completed.
I1211 15:15:56.889821 2903826 main.go:103] [DEBUG] check success chunks...
I1211 15:15:56.889840 2903826 main.go:111] [DEBUG] send get request to http://localhost:8099/bff/versioneddataset/files/chunks?bucket=abc&bucketPath=def&etag=70b40381f8881546c265afe4f279cd01-2&fileName=tmp.tar.gz&md5=7ed8b2a8ea5798202b81eedf14dc978c
I1211 15:15:56.918185 2903826 main.go:348] [Step 2] get new uploadid
I1211 15:15:56.918208 2903826 main.go:143] [DEBUG] request new multipart uploadid...
I1211 15:15:56.918244 2903826 main.go:155] [DEBUG] send post request to http://localhost:8099/bff/versioneddataset/files/chunks, with body {"chunkCount":2,"size":33554432,"md5":"7ed8b2a8ea5798202b81eedf14dc978c","fileName":"tmp.tar.gz","bucket":"abc","bucketPath":"def"}...
I1211 15:15:56.948222 2903826 main.go:359] [Step 3] tart uploading files based on uploadid NTg5MTkzYmMtNGNiYS00M2ExLWJiNDYtNjUyMDNkNDQyZjFkLmRkYWE1ZDA5LTUxZjUtNGUzNS05Yjg3LTQyZmRiMzVjOGE4MQ.
I1211 15:15:56.948298 2903826 main.go:190] [DEBUG] request upload url by uploadid: NTg5MTkzYmMtNGNiYS00M2ExLWJiNDYtNjUyMDNkNDQyZjFkLmRkYWE1ZDA5LTUxZjUtNGUzNS05Yjg3LTQyZmRiMzVjOGE4MQ...
I1211 15:15:56.948343 2903826 main.go:190] [DEBUG] request upload url by uploadid: NTg5MTkzYmMtNGNiYS00M2ExLWJiNDYtNjUyMDNkNDQyZjFkLmRkYWE1ZDA5LTUxZjUtNGUzNS05Yjg3LTQyZmRiMzVjOGE4MQ...
I1211 15:15:56.948368 2903826 main.go:202] [DEBUG] send post request to http://localhost:8099/bff/versioneddataset/files/chunk_url, with body {"partNumber":2,"size":33554432,"md5":"7ed8b2a8ea5798202b81eedf14dc978c","uploadID":"NTg5MTkzYmMtNGNiYS00M2ExLWJiNDYtNjUyMDNkNDQyZjFkLmRkYWE1ZDA5LTUxZjUtNGUzNS05Yjg3LTQyZmRiMzVjOGE4MQ","bucket":"abc","bucketPath":"def"}...
I1211 15:15:56.948383 2903826 main.go:202] [DEBUG] send post request to http://localhost:8099/bff/versioneddataset/files/chunk_url, with body {"partNumber":1,"size":33554432,"md5":"7ed8b2a8ea5798202b81eedf14dc978c","uploadID":"NTg5MTkzYmMtNGNiYS00M2ExLWJiNDYtNjUyMDNkNDQyZjFkLmRkYWE1ZDA5LTUxZjUtNGUzNS05Yjg3LTQyZmRiMzVjOGE4MQ","bucket":"abc","bucketPath":"def"}...
I1211 15:15:57.987389 2903826 main.go:389] [Step 4], all chunks are uploaded successfully and merging of chunks begins.
I1211 15:15:57.987408 2903826 main.go:231] [DEBUG] all chunks are uploaded, merge all chunks...
I1211 15:15:57.987456 2903826 main.go:242] [DEBUG] send put request to http://localhost:8099/bff/versioneddataset/files/chunks, with body {"md5":"7ed8b2a8ea5798202b81eedf14dc978c","bucket_path":"def","bucket":"abc","file_name":"tmp.tar.gz","uploadID":"NTg5MTkzYmMtNGNiYS00M2ExLWJiNDYtNjUyMDNkNDQyZjFkLmRkYWE1ZDA5LTUxZjUtNGUzNS05Yjg3LTQyZmRiMzVjOGE4MQ"}...
I1211 15:15:58.089296 2903826 main.go:406] [Step 5], Congratulations, the file was uploaded successfully



# run again, Because the same file already exists, it will not be uploaded again, whether the file is the same or not is calculated by etag.
./main --file=tmp.tar.gz --bucket=abc --bucket-path=def

I1211 15:16:22.142334 2903929 main.go:276] [DEBUG] ***** part 0, md5 is 8e3192f72fba1faee864edaa6f1636fc
I1211 15:16:22.198029 2903929 main.go:276] [DEBUG] ***** part 1, md5 is 29ee8a4e3f980fb9fe8b572c39d3caea
I1211 15:16:22.198065 2903929 main.go:323] [DEBUG] file md5 7ed8b2a8ea5798202b81eedf14dc978c, etag: 70b40381f8881546c265afe4f279cd01-2...
I1211 15:16:22.198085 2903929 main.go:331] [Step 1] check the number of chunks the file has completed.
I1211 15:16:22.198098 2903929 main.go:103] [DEBUG] check success chunks...
I1211 15:16:22.198117 2903929 main.go:111] [DEBUG] send get request to http://localhost:8099/bff/versioneddataset/files/chunks?bucket=abc&bucketPath=def&etag=70b40381f8881546c265afe4f279cd01-2&fileName=tmp.tar.gz&md5=7ed8b2a8ea5798202b81eedf14dc978c
I1211 15:16:22.200152 2903929 main.go:339] [Done], the file already exists and does not need to be uploaded again
```

2. Download the file you just uploaded

```shell
./main --action=download --file=tmp.tar.gz --bucket-path=def --bucket=abc

I1211 15:16:42.701067 2904044 main.go:410] [Step 1] get file size
I1211 15:16:42.703150 2904044 main.go:436] [DEBUG] file size is 51392595
I1211 15:16:42.703168 2904044 main.go:440] [Step 2] create local file tmp.gz
I1211 15:16:42.703225 2904044 main.go:461] [Step 3] start to donwload...
I1211 15:16:42.703303 2904044 main.go:482] [Chunk 41943040-51392595] send request to http://localhost:8099/bff/versioneddataset/files/download?bucket=abc&bucketPath=def&end=51392595&fileName=tmp.tar.gz&from=41943040
I1211 15:16:42.703391 2904044 main.go:482] [Chunk 10485760-20971520] send request to http://localhost:8099/bff/versioneddataset/files/download?bucket=abc&bucketPath=def&end=20971520&fileName=tmp.tar.gz&from=10485760
I1211 15:16:42.703452 2904044 main.go:482] [Chunk 20971520-31457280] send request to http://localhost:8099/bff/versioneddataset/files/download?bucket=abc&bucketPath=def&end=31457280&fileName=tmp.tar.gz&from=20971520
I1211 15:16:42.703574 2904044 main.go:482] [Chunk 0-10485760] send request to http://localhost:8099/bff/versioneddataset/files/download?bucket=abc&bucketPath=def&end=10485760&fileName=tmp.tar.gz&from=0
I1211 15:16:42.704383 2904044 main.go:482] [Chunk 31457280-41943040] send request to http://localhost:8099/bff/versioneddataset/files/download?bucket=abc&bucketPath=def&end=41943040&fileName=tmp.tar.gz&from=31457280
I1211 15:16:42.990110 2904044 main.go:502] [Step 4] File download complete
```

### How to calculate etag

Refer to function `fileMD5andEtag`
