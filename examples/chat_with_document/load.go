/*
Copyright 2023 The KubeAGI Authors.

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

package main

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/textsplitter"
	"os"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/documentloaders"

	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/vectorstores/chromadb"
)

var (
	path         string
	chunkSize    int
	chunkOverlap int
	loadPDF      bool
)

func NewLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load [usage]",
		Short: "Load a document",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoad(context.Background())
		},
	}

	cmd.Flags().StringVar(&apiKey, "apikey", "", "used to connect to ZhiPuAI platform(required)")
	cmd.Flags().StringVar(&url, "vector-store", "", "the chromaDB vector database url(required)")
	cmd.Flags().StringVar(&path, "path", "", "the document path")
	cmd.Flags().StringVar(&namespace, "namespace", _defaultNamespace, "the vector database namespace")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", 4000, "the chunk size")
	cmd.Flags().IntVar(&chunkOverlap, "chunk-overlap", 200, "the chunk overlap")
	cmd.Flags().BoolVar(&loadPDF, "load-pdf", false, "load pdf files")

	if err := cmd.MarkFlagRequired("apikey"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("vector-store"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("path"); err != nil {
		panic(err)
	}

	return cmd
}

func runLoad(ctx context.Context) error {
	fmt.Println("Loading documents...")

	// check ZhiPuAI api key, build embedder
	if apiKey == "" {
		return fmt.Errorf("ZhiPuAI api key is empty")
	}
	if url == "" {
		return fmt.Errorf("chromaDB scheme is empty")
	}

	fmt.Println("Connecting platform...")
	z := zhipuai.NewZhiPuAI(apiKey)
	_, err := z.Validate()
	if err != nil {
		return fmt.Errorf("error validating ZhiPuAI api key: %s", err.Error())
	}

	embedder, err := zhipuaiembeddings.NewZhiPuAI(
		zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
	)
	if err != nil {
		return fmt.Errorf("error creating ZhiPuAI embedder: %s", err.Error())
	}

	fmt.Println("Connecting vector database...")
	_, err = chromadb.New(
		chromadb.WithURL(url),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(namespace),
	)
	if err != nil {
		return fmt.Errorf("error connecting chroma db: %s", err.Error())
	}

	// load document
	if loadPDF {
		return loadPDFFiles(ctx, path)
	}
	return loadAndStoreFiles(ctx, path)
}

func loadAndStoreFiles(ctx context.Context, path string) error {
	files, err := getAllFilesFromPath(path)
	if err != nil {
		return fmt.Errorf("error getting files from path %s: %s", path, err.Error())
	}
	fmt.Printf("%v files to be loaded...\n", len(files))

	for _, f := range files {
		fmt.Printf("Reading file: %s\n", f)
		// check file type
		source, rErr := os.ReadFile(f)
		if rErr != nil {
			return fmt.Errorf("error reading file: %s", rErr.Error())
		}
		lErr := loadAndSplit(ctx, string(source))
		if lErr != nil {
			return fmt.Errorf("error loading file: %s", lErr.Error())
		}
	}

	return nil
}

func getAllFilesFromPath(path string) ([]string, error) {
	var results []string

	// TODO: can this be optimized?
	// check if path refers to a file or a directory
	if isDir(path) {
		// get all dirEntry inside the path
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, p := range entries {
			if p.IsDir() {
				// recursively load all files
				paths, err := getAllFilesFromPath(path + "/" + p.Name())
				if err != nil {
					return nil, err
				}
				results = append(results, paths...)
			} else {
				// build real path to file
				realPath := path + "/" + p.Name()
				results = append(results, realPath)
			}
		}
	} else {
		results = append(results, path)
	}

	return results, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func loadAndSplit(ctx context.Context, sourceFile string) error {

	workload := Workload{
		Document:     sourceFile,
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
	}

	return workload.EmbedAndStoreDocument(ctx)
}

func loadPDFFiles(ctx context.Context, path string) error {
	if isDir(path) {
		return fmt.Errorf("path %s is a directory", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file: %s", err.Error())
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %s", err.Error())
	}

	splitter := textsplitter.NewRecursiveCharacter()

	p := documentloaders.NewPDF(f, fInfo.Size())
	docs, err := p.LoadAndSplit(context.Background(), splitter)

	embedder, err := zhipuaiembeddings.NewZhiPuAI(
		zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
	)
	if err != nil {
		return fmt.Errorf("error creating ZhiPuAI embedder: %s", err.Error())
	}

	db, err := chromadb.New(
		chromadb.WithURL(url),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(namespace),
	)
	if err != nil {
		return fmt.Errorf("error connecting chroma db: %s", err.Error())
	}

	fmt.Printf("Adding %v documents...\n", len(docs))

	return db.AddDocuments(ctx, docs)
}
