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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/client"
)

var (
	err  error
	home string

	namespace string

	kubeClient dynamic.Interface
)

func NewCLI() *cobra.Command {
	arctl := &cobra.Command{
		Use:   "arctl [usage]",
		Short: "Command line tools for Arcadia",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(home); os.IsNotExist(err) {
				if err := os.MkdirAll(home, 0700); err != nil {
					return err
				}
			}

			// initialize a kube client
			kubeClient, err = client.GetClient(nil)
			if err != nil {
				return err
			}

			return nil
		},
	}

	arctl.AddCommand(NewDatasourceCmd())
	arctl.AddCommand(NewDatasetCmd())
	arctl.AddCommand(NewChatCmd())

	arctl.PersistentFlags().StringVar(&home, "home", filepath.Join(os.Getenv("HOME"), ".arcadia"), "home directory to use")
	arctl.PersistentFlags().StringVar(&namespace, "namespace", "default", "namespace to use")

	return arctl
}

func main() {
	if err := NewCLI().Execute(); err != nil {
		panic(err)
	}
}
