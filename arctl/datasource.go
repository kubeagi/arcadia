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

	"github.com/spf13/cobra"

	"github.com/kubeagi/arcadia/arctl/printer"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/datasource"
)

var (
	datasourcePrintHeaders = []string{"name", "displayName", "creator", "endpoint"}
)

func NewDatasourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "datasource [usage]",
		Short: "Manage datasources",
	}

	cmd.AddCommand(DatasourceListCmd())

	return cmd
}

func DatasourceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [usage]",
		Short: "List many datasources",
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(os.Args) == 4 {
				name = os.Args[3]
			}

			list, err := datasource.DatasourceList(cmd.Context(), kubeClient, name, namespace, "", "")
			if err != nil {
				return err
			}
			objects := make([]any, len(list))
			for i, item := range list {
				objects[i] = *item
			}
			printer.Print(datasourcePrintHeaders, objects)
			return nil
		},
	}

	return cmd
}
