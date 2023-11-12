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

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	chromago "github.com/amikos-tech/chroma-go"
	chromaopenapi "github.com/amikos-tech/chroma-go/swagger"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	openaiEmbeddings "github.com/tmc/langchaingo/embeddings/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/chroma"
	"k8s.io/klog/v2"

	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

var (
	dataset string

	llmType string
	apiKey  string

	// path to documents seperated by comma
	inputDocuments string

	vectorStore      string
	documentLanguage string
	textSplitter     string
	chunkSize        int
	chunkOverlap     int

	// force remove
	resetVectorStore bool
)

func NewDatasetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dataset [usage]",
		Short: "Manage dataset locally",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			datasetDir := filepath.Join(home, "dataset")
			if _, err := os.Stat(datasetDir); os.IsNotExist(err) {
				if err := os.MkdirAll(datasetDir, 0700); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.AddCommand(DatasetListCmd())
	cmd.AddCommand(DatasetCreateCmd())
	cmd.AddCommand(DatasetShowCmd())
	cmd.AddCommand(DatasetExecuteCmd())
	cmd.AddCommand(DatasetDeleteCmd())

	return cmd
}

// DatasetListCmd returns a Cobra command for listing datasets.
func DatasetListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [usage]",
		Short: "List dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("| DATASET | FILES |EMBEDDING MODEL | VECTOR STORE | DOCUMENT LANGUAGE | TEXT SPLITTER | CHUNK SIZE | CHUNK OVERLAP |\n")
			err = filepath.Walk(filepath.Join(home, "dataset"), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// skip directory
				if info.IsDir() {
					return nil
				}
				ds, err := loadCachedDataset(path)
				if err != nil {
					return fmt.Errorf("failed to load cached dataset %s: %v", info.Name(), err)
				}
				// print item
				fmt.Printf("| %s | %d | %s | %s | %s | %s | %d | %d |\n", ds.Name, len(ds.Files), ds.LLMType, ds.VectorStore, ds.DocumentLanguage, ds.TextSplitter, ds.ChunkSize, ds.ChunkOverlap)
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

// DatasetCreateCmd returns a new instance of the cobra.Command that is used to create a dataset.
func DatasetCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [usage]",
		Short: "Create dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.Infof("Create dataset: %s \n", dataset)
			ds, err := loadCachedDataset(filepath.Join(home, "dataset", dataset))
			if err != nil {
				return err
			}
			if ds.Name != "" {
				return fmt.Errorf("dataset %s already exists", dataset)
			}
			// set dataset
			ds.Name = dataset
			ds.CreateTime = time.Now().String()
			ds.LLMApiKey = apiKey
			ds.LLMType = llmType
			ds.VectorStore = vectorStore
			ds.DocumentLanguage = documentLanguage
			ds.TextSplitter = textSplitter
			ds.ChunkSize = chunkSize
			ds.ChunkOverlap = chunkOverlap

			err = ds.execute(context.Background())
			if err != nil {
				// only print error but do not exit
				klog.Errorf("failed to execute dataset %s: %v", dataset, err)
			}

			// cache the dataset to local
			cache, err := json.Marshal(ds)
			if err != nil {
				return fmt.Errorf("failed to marshal dataset %s: %v", dataset, err)
			}
			err = os.WriteFile(filepath.Join(home, "dataset", dataset), cache, 0644)
			if err != nil {
				return err
			}
			klog.Infof("Successfully created dataset %s\n", dataset)

			return showDataset(dataset)
		},
	}
	cmd.Flags().StringVar(&dataset, "name", "", "dataset(namespace/collection) of the document to load into")
	if err = cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&inputDocuments, "documents", "", "path of the documents/document directories to load(separated by comma and directories supported)")
	if err = cmd.MarkFlagRequired("documents"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&llmType, "llm-type", string(llms.ZhiPuAI), "llm type to use(Only zhipuai,openai supported now)")
	cmd.Flags().StringVar(&apiKey, "llm-apikey", "", "apiKey to access embedding service")
	if err = cmd.MarkFlagRequired("llm-apikey"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&vectorStore, "vector-store", "http://127.0.0.1:8000", "vector stores to use(Only chroma supported now)")
	cmd.Flags().StringVar(&documentLanguage, "document-language", "text", "language of the document(Only text,html,csv supported now)")
	cmd.Flags().StringVar(&textSplitter, "text-splitter", "character", "text splitter to use(Only character,token,markdown supported now)")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", 300, "chunk size for embedding")
	cmd.Flags().IntVar(&chunkOverlap, "chunk-overlap", 30, "chunk overlap for embedding")

	return cmd
}

func DatasetShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [usage]",
		Short: "Load more documents to dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.Infof("Show dataset: %s \n", dataset)
			return showDataset(dataset)
		},
	}

	cmd.Flags().StringVar(&dataset, "name", "", "dataset(namespace/collection) of the document to load into")
	if err = cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	return cmd
}

func showDataset(dataset string) error {
	cachedDatasetFile, err := os.OpenFile(filepath.Join(home, "dataset", dataset), os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			klog.Errorf("dataset %s does not exist", dataset)
			return nil
		} else {
			return fmt.Errorf("failed to open cached dataset file: %v", err)
		}
	}
	defer cachedDatasetFile.Close()

	data, err := io.ReadAll(cachedDatasetFile)
	if err != nil {
		return fmt.Errorf("failed to read cached dataset file: %v", err)
	}
	// Create a buffer to store the formatted JSON
	var formattedJSON bytes.Buffer

	// Indent and format the JSON
	err = json.Indent(&formattedJSON, data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format cached dataset file: %v", err)
	}

	// print dataset
	klog.Infof("\n%s", formattedJSON.String())

	return nil
}

func DatasetExecuteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute [usage]",
		Short: "Execute dataset to load documents to dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.Infof("Execute dataset: %s \n", dataset)
			ds, err := loadCachedDataset(filepath.Join(home, "dataset", dataset))
			if err != nil {
				return err
			}
			if ds.Name == "" {
				return fmt.Errorf("dataset %s does not exist", dataset)
			}

			err = ds.execute(context.Background())
			if err != nil {
				// only print error but do not exit
				klog.Errorf("failed to execute dataset %s: %v", dataset, err)
			}

			// cache the dataset to local
			klog.Infof("Caching dataset %s", dataset)
			cache, err := json.Marshal(ds)
			if err != nil {
				return fmt.Errorf("failed to marshal dataset %s: %v", dataset, err)
			}
			err = os.WriteFile(filepath.Join(home, "dataset", dataset), cache, 0644)
			if err != nil {
				return err
			}
			klog.Infof("Successfully execute dataset: %s \n", dataset)
			return nil
		},
	}

	cmd.Flags().StringVar(&dataset, "name", "", "dataset(namespace/collection) of the document to load into")
	if err = cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&inputDocuments, "documents", "", "path of the documents/document directories to load(separated by comma and directories supported)")
	if err = cmd.MarkFlagRequired("documents"); err != nil {
		panic(err)
	}

	return cmd
}
func DatasetDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [usage]",
		Short: "Delete dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.Infof("Delete dataset: %s \n", dataset)
			ds, err := loadCachedDataset(filepath.Join(home, "dataset", dataset))
			if err != nil {
				return fmt.Errorf("failed to load cached dataset %s: %v", dataset, err)
			}
			if ds.Name == "" {
				klog.Infof("Dataset %s does not exist", dataset)
				return nil
			}

			// remove dateset from remote vector store
			if resetVectorStore {
				configuration := chromaopenapi.NewConfiguration()
				configuration.Servers = chromaopenapi.ServerConfigurations{
					{
						URL:         ds.VectorStore,
						Description: "chroma server url for this store",
					},
				}
				client := &chromago.Client{
					ApiClient: chromaopenapi.NewAPIClient(configuration),
				}
				_, err := client.Reset()
				if err != nil {
					return err
				}
			}
			// remove local cache
			if err := os.Remove(filepath.Join(home, "dataset", dataset)); err != nil {
				panic(err)
			}
			klog.Infof("Successfully delete dataset: %s \n", dataset)
			return nil
		},
	}

	cmd.Flags().StringVar(&dataset, "name", "arcadia", "dataset(namespace/collection) of the document to load into")
	if err = cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	cmd.Flags().BoolVar(&resetVectorStore, "reset-vector-store", false, "forcely reset dataset from remote vector store")

	return cmd
}

type Dataset struct {
	Name       string `json:"name"`
	CreateTime string `json:"create_time"`

	// Parameters for embedding service
	LLMType   string `json:"llm_type"`
	LLMApiKey string `json:"llm_api_key"`

	// Parameters for vectorization
	VectorStore      string `json:"vector_store"`
	DocumentLanguage string `json:"document_language"`
	TextSplitter     string `json:"text_splitter"`
	ChunkSize        int    `json:"chunk_size"`
	ChunkOverlap     int    `json:"chunk_overlap"`

	Files map[string]File `json:"files"`
}

type File struct {
	// basic info
	Path string `json:"path"`
	Size int64  `json:"size"`

	// embedding status
	// Chunks is the number of splitted chunks
	Chunks int `json:"chunks"`
	// ChunksLoaded is the number of chunks loaded
	ChunksLoaded int `json:"chunks_loaded"`

	TimeCost float64 `json:"time_cost"`
}

func loadCachedDataset(cachedDatasetFilePath string) (*Dataset, error) {
	cachedDatasetFile, err := os.OpenFile(cachedDatasetFilePath, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			cachedDatasetFile, err = os.Create(cachedDatasetFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to create cached dataset file: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to open/create cached dataset file: %v", err)
		}
	}
	defer cachedDatasetFile.Close()

	content, err := io.ReadAll(cachedDatasetFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached dataset file: %v", err)
	}
	ds := &Dataset{
		Files: map[string]File{},
	}
	if len(content) != 0 {
		err = json.Unmarshal(content, ds)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal cached dataset file: %v", err)
		}
	}
	return ds, nil
}

// LoadDocuments loads documents to vector store.
func (cachedDS *Dataset) execute(ctx context.Context) error {
	for _, document := range strings.Split(inputDocuments, ",") {
		fileInfo, err := os.Stat(document)
		if err != nil {
			return err
		}
		// load documents under a document directory
		if fileInfo.IsDir() {
			if err = filepath.Walk(document, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// skip if it is a directory
				if info.IsDir() {
					return nil
				}
				// process documents
				klog.Infof("Loading document: %s \n", path)
				return cachedDS.loadDocument(ctx, path)
			}); err != nil {
				return err
			}
		} else {
			klog.Infof("Loading document: %s \n", document)
			err := cachedDS.loadDocument(ctx, document)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// LoadDocument loads a document from a file and splits it into multiple documents.
func (cachedDS *Dataset) loadDocument(ctx context.Context, document string) error {
	start := time.Now()
	// read document
	file, err := os.Open(document)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file content: %v", err)
	}

	// skip if all chunks has been loaded
	hash := sha256.New()
	hash.Write(data)
	digest := hash.Sum(nil)
	cachedFile, ok := cachedDS.Files[hex.EncodeToString(digest)]
	// TODO: check if cached.Chunks == cached.ChunksLoaded
	if ok && cachedFile.Chunks == cachedFile.ChunksLoaded {
		klog.Infof("Document %s has been loaded.Skip loading", document)
		return nil
	}

	dataReader := bytes.NewReader(data)
	var loader documentloaders.Loader
	switch documentLanguage {
	case "text":
		loader = documentloaders.NewText(dataReader)
	case "csv":
		loader = documentloaders.NewCSV(dataReader)
	case "html":
		loader = documentloaders.NewHTML(dataReader)
	default:
		return errors.New("unsupported document language")
	}

	// initliaze text splitter
	var split textsplitter.TextSplitter
	switch cachedDS.TextSplitter {
	case "token":
		split = textsplitter.NewTokenSplitter(
			textsplitter.WithChunkSize(chunkSize),
			textsplitter.WithChunkOverlap(chunkOverlap),
		)
	case "markdown":
		split = textsplitter.NewMarkdownTextSplitter(
			textsplitter.WithChunkSize(chunkSize),
			textsplitter.WithChunkOverlap(chunkOverlap),
		)
	default:
		split = textsplitter.NewRecursiveCharacter(
			textsplitter.WithChunkSize(chunkSize),
			textsplitter.WithChunkOverlap(chunkOverlap),
		)
	}

	documents, err := loader.LoadAndSplit(ctx, split)
	if err != nil {
		return err
	}

	err = cachedDS.embedDocuments(context.Background(), documents)
	if err != nil {
		return err
	}

	// cache the document
	fileInfo, err := os.Stat(document)
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	cacheFile := File{
		Path:         document,
		Size:         fileInfo.Size(),
		Chunks:       len(documents),
		ChunksLoaded: len(documents),
		TimeCost:     time.Since(start).Seconds(),
	}
	cachedDS.Files[hex.EncodeToString(digest)] = cacheFile

	klog.Infof("Time cost %.2f seconds for loading document: %s \n", cacheFile.TimeCost, document)
	return nil
}

func (cachedDS *Dataset) embedDocuments(ctx context.Context, documents []schema.Document) error {
	var embedder embeddings.Embedder
	var err error

	switch llmType {
	case "zhipuai":
		embedder, err = zhipuaiembeddings.NewZhiPuAI(
			zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(cachedDS.LLMApiKey)),
		)
		if err != nil {
			return err
		}
	case "openai":
		embedder, err = openaiEmbeddings.NewOpenAI()
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported embedding type")
	}

	chroma, err := chroma.New(
		chroma.WithChromaURL(cachedDS.VectorStore),
		chroma.WithEmbedder(embedder),
		chroma.WithNameSpace(cachedDS.Name),
	)
	if err != nil {
		return err
	}

	return chroma.AddDocuments(ctx, documents)
}
