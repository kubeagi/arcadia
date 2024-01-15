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

package arctl

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	basev1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/pkg/evaluation"
)

func NewEvalCmd(home *string, namespace *string) *cobra.Command {
	var appName string

	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Manage evaluations",
	}

	cmd.PersistentFlags().StringVar(&appName, "application", "", "The application to be evaluated")
	if err := cmd.MarkPersistentFlagRequired("application"); err != nil {
		panic(err)
	}

	cmd.AddCommand(EvalGenTestDataset(home, namespace, &appName))

	return cmd
}

func EvalGenTestDataset(home *string, namespace *string, appName *string) *cobra.Command {
	var inputDir string
	var questionColumn string
	var groundTruthsColumn string
	var outputMethod string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "gen_test_dataset",
		Short: "Generate a test dataset for evaluation",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if outputDir == "" {
				outputDir = *home
			}

			// init kubeclient
			kubeClient, err := client.GetClient(nil)
			if err != nil {
				return err
			}

			// read files
			app := &basev1alpha1.Application{}
			obj, err := common.ResouceGet(ctx, kubeClient, generated.TypedObjectReferenceInput{
				APIGroup:  &common.ArcadiaAPIGroup,
				Kind:      "Application",
				Namespace: namespace,
				Name:      *appName,
			}, v1.GetOptions{})
			if err != nil {
				return err
			}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), app)
			if err != nil {
				return err
			}

			// read files from input directory
			files, err := os.ReadDir(inputDir)
			if err != nil {
				log.Fatal(err)
			}
			for _, file := range files {
				if file.IsDir() || filepath.Ext(file.Name()) != ".csv" || strings.HasPrefix(file.Name(), "ragas-") {
					continue
				}
				var output evaluation.Output
				switch outputMethod {
				case "csv":
					outputCSVFile, err := os.Create(filepath.Join(inputDir, fmt.Sprintf("ragas-%s", file.Name())))
					if err != nil {
						return err
					}
					defer outputCSVFile.Close()
					csvOutput := &evaluation.CSVOutput{
						W: csv.NewWriter(outputCSVFile),
					}
					defer csvOutput.W.Flush()
					output = csvOutput
				default:
					output = &evaluation.PrintOutput{}
				}
				// read file from dataset
				err = GenDatasetOnSingleFile(ctx, kubeClient, app,
					filepath.Join(inputDir, file.Name()),
					evaluation.WithQuestionColumn(questionColumn),
					evaluation.WithGroundTruthsColumn(groundTruthsColumn),
					evaluation.WithOutput(output),
				)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&inputDir, "input-dir", "", "The input directory where to load original dataset files")
	if err := cmd.MarkFlagRequired("input-dir"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&questionColumn, "question-column", "q", "The column name which provides questions")
	cmd.Flags().StringVar(&groundTruthsColumn, "ground-truths-column", "a", "The column name which provides the answers")
	cmd.Flags().StringVar(&outputMethod, "output", "", "The way to output the generated dataset rows.We support two ways: \n - stdout: print row \n - csv: save row to csv file")

	return cmd
}

func GenDatasetOnSingleFile(ctx context.Context, kubeClient dynamic.Interface, app *basev1alpha1.Application, file string, genOpts ...evaluation.GenOptions) error {
	// read file content
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// init evaluation dataset generator
	generator, err := evaluation.NewRagasDatasetGenerator(ctx, kubeClient, app, genOpts...)
	if err != nil {
		return err
	}

	// generate test dataset
	err = generator.Generate(
		ctx,
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}

	return nil
}
