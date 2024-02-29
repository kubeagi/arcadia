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

package documentloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	documentloaders "github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/app-node/documentloader/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	arcadiadocumentloaders "github.com/kubeagi/arcadia/pkg/documentloaders"
)

type DocumentLoader struct {
	base.BaseNode
	Instance *v1alpha1.DocumentLoader
}

func NewDocumentLoader(baseNode base.BaseNode) *DocumentLoader {
	return &DocumentLoader{
		BaseNode: baseNode,
	}
}

func (dl *DocumentLoader) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &v1alpha1.DocumentLoader{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: dl.RefNamespace(), Name: dl.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the documentloader in cluster: %w", err)
	}
	dl.Instance = instance
	return nil
}

func (dl *DocumentLoader) Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error) {
	// Check if have docs as input
	v1, ok := args["files"]
	if !ok {
		return args, errors.New("no input docs")
	}
	files, ok := v1.([]string)
	if !ok || len(files) == 0 {
		return args, errors.New("empty file list")
	}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: dl.RefNamespace(), Name: dl.Ref.Name}, dl.Instance); err != nil {
		return args, fmt.Errorf("can't find the documentloader in cluster: %w", err)
	}

	system, err := config.GetSystemDatasource(ctx, cli)
	if err != nil {
		return nil, err
	}
	endpoint := system.Spec.Endpoint.DeepCopy()
	if endpoint != nil && endpoint.AuthSecret != nil {
		endpoint.AuthSecret.WithNameSpace(system.Namespace)
	}
	ossDatasource, err := datasource.NewLocal(ctx, cli, endpoint)
	if err != nil {
		return nil, err
	}

	var allDocs []schema.Document
	var textArray []string

	for _, file := range files {
		ossInfo := &arcadiav1alpha1.OSS{Bucket: dl.RefNamespace()}
		ossInfo.Object = filepath.Join("upload", file)
		klog.Infoln("handling file", ossInfo.Object)
		fileHandler, err := ossDatasource.ReadFile(ctx, ossInfo)
		if err != nil {
			klog.Errorln("failed to handle file", err)
			continue
		}
		defer fileHandler.Close()
		// TODO: cache the content if the hash does not change, and use the content directly without read and load it again

		data, err := io.ReadAll(fileHandler)
		if err != nil {
			klog.Errorln("failed to read file content", err)
			continue
		}

		var loader documentloaders.Loader
		// Use ext name in the spec first, and use real file ext name if it does not exist
		extName := dl.Instance.Spec.FileExtName
		if extName == "" {
			extName = filepath.Ext(file)
		}
		switch extName {
		case ".mp3", ".wav":
			loader = arcadiadocumentloaders.NewAudoWithWhisper(data, file, "", "", false)
		case ".csv":
			dataReader := bytes.NewReader(data)
			loader = documentloaders.NewCSV(dataReader)
		case ".html", ".htm":
			dataReader := bytes.NewReader(data)
			loader = documentloaders.NewHTML(dataReader)
		case ".pdf":
			dataReader := bytes.NewReader(data)
			loader = documentloaders.NewPDF(dataReader, int64(len(data)))
		default:
			dataReader := bytes.NewReader(data)
			loader = documentloaders.NewText(dataReader)
		}

		split := textsplitter.NewRecursiveCharacter(
			textsplitter.WithChunkSize(dl.Instance.Spec.ChunkSize),
			textsplitter.WithChunkOverlap(dl.Instance.Spec.ChunkOverlap),
		)
		docs, err := loader.LoadAndSplit(ctx, split)
		if err != nil {
			klog.Errorln("failed to load and split content", err)
			return nil, err
		}
		allDocs = append(allDocs, docs...)
		for _, doc := range docs {
			textArray = append(textArray, doc.PageContent)
		}
	}

	// Set both docs and context for latter usage
	args["docs"] = allDocs
	args["context"] = strings.Join(textArray, "\n")
	return args, nil
}

func (dl *DocumentLoader) Ready() (isReady bool, msg string) {
	// TODO: use instance.Status.IsReadyOrGetReadyMessage() later if needed
	return true, ""
}
