package dashscope

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type EmbeddingRequest struct {
	Model      Model               `json:"model"`
	Input      EmbeddingInput      `json:"input"`
	Parameters EmbeddingParameters `json:"parameters"`
}

type EmbeddingInput struct {
	*EmbeddingInputSync
	*EmbeddingInputAsync
}
type EmbeddingInputSync struct {
	Texts []string `json:"texts,omitempty"`
}
type EmbeddingInputAsync struct {
	URL string `json:"url,omitempty"`
}
type EmbeddingParameters struct {
	TextType TextType `json:"text_type"`
}

type TextType string

const (
	TextTypeQuery    TextType = "query"
	TextTypeDocument TextType = "document"
)

type EmbeddingResponse struct {
	CommonResponse
	Output EmbeddingOutput `json:"output"`
	Usage  EmbeddingUsage  `json:"usage"`
}

type EmbeddingOutput struct {
	*EmbeddingOutputSync
	*EmbeddingOutputASync
}
type EmbeddingOutputSync struct {
	Embeddings []Embeddings `json:"embeddings"`
}

type EmbeddingOutputASync struct {
	TaskID        string     `json:"task_id"`
	TaskStatus    TaskStatus `json:"task_status"`
	URL           string     `json:"url"`
	SubmitTime    string     `json:"submit_time,omitempty"`
	ScheduledTime string     `json:"scheduled_time,omitempty"`
	EndTime       string     `json:"end_time,omitempty"`
	// when failed
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
type Embeddings struct {
	TextIndex int       `json:"text_index"`
	Embedding []float64 `json:"embedding"`
}

type EmbeddingUsage struct {
	TotalTokens int `json:"total_tokens"`
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "PENDING"
	TaskStatusRunning   TaskStatus = "RUNNING"
	TaskStatusSucceeded TaskStatus = "SUCCEEDED"
	TaskStatusFailed    TaskStatus = "FAILED"
	TaskStatusUnknown   TaskStatus = "UNKNOWN"
)

func DownloadAndExtract(url string, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	_, err = io.Copy(destFile, gzReader)
	if err != nil {
		return err
	}
	return nil
}
