# Build the manager binary
ARG GO_VER=1.21
FROM golang:${GO_VER} as builder
ARG GOPROXY=https://goproxy.cn,direct
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go env -w GOPROXY=${GOPROXY}
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY apiserver/ apiserver/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apiserver-bin apiserver/main.go

# Use alpine as minimal base image to package the manager binary
FROM alpine:3.19.1

RUN apk update \
    # Install packages to support pdf to text conversion
    && apk add --no-cache  poppler-utils wv unrtf tidyhtml

WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/apiserver-bin ./apiserver

RUN adduser -D -u 1000 1000

USER 1000

ENTRYPOINT ["/manager"]
