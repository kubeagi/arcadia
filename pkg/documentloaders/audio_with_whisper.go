/*
Copyright 2024 KubeAGI.

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

package documentloaders

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"k8s.io/klog/v2"
)

// Use the whisper API to transcribe the audio
const _url = "http://whisper-apiserver.kubeagi-system:9000/asr"

// AudioWithWhisper represents a audio using whisper document loader.
type AudioWithWhisper struct {
	fileName  string
	data      []byte
	vadFilter bool
	language  string
	output    string
}

var _ documentloaders.Loader = AudioWithWhisper{}

// NewAudoWithWhisper creates a new audo loader with an io.Reader and optional column names for filtering.
func NewAudoWithWhisper(data []byte, fileName, language, output string, vadFilter bool) AudioWithWhisper {
	q := AudioWithWhisper{
		fileName:  fileName,
		data:      data,
		language:  language,
		vadFilter: vadFilter,
		output:    output,
	}
	return q
}

// Load reads from the io.Reader and returns a document with the data.
func (aww AudioWithWhisper) Load(ctx context.Context) ([]schema.Document, error) {
	// TODO: Update the status of document loader CR

	// Run the whisperloader tool to transcribe audio to text
	text, err := aww.CovertToText(ctx)
	if err != nil {
		return nil, err
	}
	return []schema.Document{
		{
			PageContent: text,
			Metadata:    map[string]any{},
		},
	}, nil
}

// LoadAndSplit reads text data from the io.Reader and splits it into multiple
// documents using a text splitter.
func (aww AudioWithWhisper) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	return aww.Load(ctx)
}

// input will be from the previous LLM
func (aww AudioWithWhisper) CovertToText(ctx context.Context) (string, error) {
	klog.Infof("input for whisper api is %s, %s, %s", aww.fileName, aww.language, aww.output)

	// write data to local file
	tmpFilePath := fmt.Sprintf("/tmp/%s", aww.fileName)
	filePath := filepath.Dir(tmpFilePath)
	if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
		klog.Errorln("failed to create directory: %s", err)
		return "", err
	}
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		klog.Errorln("failed to create file: %s", err)
		return "", err
	}
	defer tmpFile.Close()
	_, err = tmpFile.Write(aww.data)
	if err != nil {
		klog.Errorln("failed to write file: %s", err)
		return "", err
	}

	// prepare the form data
	openFile, err := os.Open(tmpFilePath)
	if err != nil {
		return "", err
	}
	defer openFile.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio_file", filepath.Base(tmpFilePath))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, openFile)
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}
	// remove the temporary file
	defer os.Remove(tmpFilePath)

	// Send to whisper API to transcribe audio
	params := make(url.Values)
	// TODO: allow to customize the params if needed
	// params.Add("language", language)
	params.Add("encode", "true")
	params.Add("task", "transcribe")
	params.Add("vad_filter", "false")
	params.Add("word_timestamps", "false")
	params.Add("output", "txt")

	reqURL := fmt.Sprintf("%s?%s", _url, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while calling whisper API: %w", err)
	}
	defer response.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, response.Body)
	if err != nil {
		return "", fmt.Errorf("error while coping data in whisper API: %w", err)
	}
	klog.Infof("result from whisper API: %s", buf.String())
	return buf.String(), nil
}
