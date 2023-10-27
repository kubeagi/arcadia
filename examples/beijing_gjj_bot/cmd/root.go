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
package cmd

import (
	"fmt"
	"os"

	"github.com/kubeagi/arcadia/examples/beijing_gjj_bot/pkg"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	apiKey       string
	dataFilePath string
	dbURL        string
	namespace    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chat_with_document",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if apiKey == "" {
			return fmt.Errorf("请通过 --apikey 输入您的apikey")
		}
		if dbURL == "" {
			return fmt.Errorf("请通过 --chromadb 输入您的chromaDB地址")
		}
		klog.Infoln("解析政策文件...")
		d, err := pkg.NewDashScope(apiKey, dbURL, namespace)
		if err != nil {
			return err
		}
		if err = d.EmbeddingFileTitle(cmd.Context(), dataFilePath); err != nil {
			return err
		}
		var input, resp string
		var node *pkg.Node
		history := make([]string, 0)
		for {
			fmt.Print("请输入（quit退出,new开启新对话）: ")
			fmt.Scanln(&input)
			if input == "quit" {
				break
			}
			if input == "new" {
				history = make([]string, 0)
				node = nil
			}
			resp, node, err = d.Query(cmd.Context(), input, history, node)
			if err != nil {
				return err
			}
			fmt.Printf("Bot: %s\n", resp)
			if resp != pkg.DontKnow {
				history = append(history, input, resp)
			}
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&apiKey, "apikey", "", "apikey for DashScope")
	rootCmd.PersistentFlags().StringVar(&dbURL, "chromadb", "", "connect URL to chromadb")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "beijing_gjj", "the vector database namespace")
	rootCmd.PersistentFlags().StringVar(&dataFilePath, "data", "./data", "data file path")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
