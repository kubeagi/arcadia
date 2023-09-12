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
	"github.com/spf13/cobra"
)

var (
	err error
)

func NewCLI() *cobra.Command {
	arctl := &cobra.Command{
		Use:   "arctl [usage]",
		Short: "Command line tools for Arcadia",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	arctl.AddCommand(NewLoadCmd())
	arctl.AddCommand(NewChatCmd())

	return arctl
}

func main() {
	if err := NewCLI().Execute(); err != nil {
		panic(err)
	}
}
