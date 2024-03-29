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

	arctlPkg "github.com/kubeagi/arcadia/pkg/arctl"
)

var (
	err  error
	home string

	namespace string
)

func NewCLI() *cobra.Command {
	arctl := &cobra.Command{
		Use:   "arctl [usage]",
		Short: "Command line tools for Arcadia",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err = os.Stat(home); os.IsNotExist(err) {
				if err := os.MkdirAll(home, 0700); err != nil {
					return err
				}
			}
			return nil
		},
	}

	arctl.PersistentFlags().StringVar(&home, "home", filepath.Join(os.Getenv("HOME"), ".arcadia"), "home directory to use")
	arctl.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "namespace to use")

	arctl.AddCommand(arctlPkg.NewDatasourceCmd(&namespace))
	arctl.AddCommand(arctlPkg.NewEvalCmd(&home, &namespace))

	return arctl
}

func main() {
	if err := NewCLI().Execute(); err != nil {
		panic(err)
	}
}
