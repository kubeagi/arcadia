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
	"fmt"

	"github.com/spf13/cobra"
)

func NewCLI() *cobra.Command {
	cli := &cobra.Command{
		Use:   "chat [usage]",
		Short: "CLI for chat server example",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cli.AddCommand(NewStartCmd())
	cli.AddCommand(NewLoadCmd())

	return cli
}

func main() {
	if err := NewCLI().Execute(); err != nil {
		fmt.Printf("Run failed, error:\n %v", err)
	}
}
