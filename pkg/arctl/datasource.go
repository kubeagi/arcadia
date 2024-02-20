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

package arctl

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/arctl/printer"
)

var (
	datasourcePrintHeaders = []string{"name", "displayName", "creator", "endpoint", "oss"}

	// common spec to all resources
	displayName string
	description string
)

func NewDatasourceCmd(namespace *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "datasource [usage]",
		Short: "Manage datasources",
	}

	cmd.AddCommand(DatasourceCreateCmd(namespace))
	cmd.AddCommand(DatasourceGetCmd(namespace))
	cmd.AddCommand(DatasourceDeleteCmd(namespace))
	cmd.AddCommand(DatasourceListCmd(namespace))

	return cmd
}

func DatasourceCreateCmd(namespace *string) *cobra.Command {
	var empytDatasource bool
	// endpoint flags
	var endpointURL, endpointAuthUser, endpointAuthPwd string
	var endpointInsecure bool
	var ossBucket string

	// command definition
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a datasource",
		Long:  "Create a datasource",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(os.Args) < 4 {
				return errors.New("missing datasource name")
			}
			name := os.Args[3]

			if endpointURL == "" && !empytDatasource {
				return errors.New("set --empty if you want to create a empty datasource")
			}

			input := generated.CreateDatasourceInput{
				Name:        name,
				Namespace:   *namespace,
				DisplayName: &displayName,
				Description: &description,
				Endpointinput: generated.EndpointInput{
					URL:      endpointURL,
					Insecure: &endpointInsecure,
				},
			}

			// create auth secret for datasource
			if endpointAuthUser != "" && endpointAuthPwd != "" {
				input.Endpointinput.Auth = map[string]interface{}{
					"rootUser":     endpointAuthUser,
					"rootPassword": endpointAuthPwd,
				}
			}

			if ossBucket != "" {
				input.Ossinput = &generated.OssInput{Bucket: ossBucket}
			}

			kubeClient, err := client.GetClient(nil)
			if err != nil {
				return err
			}
			_, err = datasource.CreateDatasource(cmd.Context(), kubeClient, input)
			if err != nil {
				return err
			}
			klog.Infof("Successfully created datasource %s\n", name)

			return nil
		},
	}
	// Common flags
	cmd.Flags().StringVar(&displayName, "display-name", "", "The displayname for datasource")
	cmd.Flags().StringVar(&description, "description", "A datasource for LLMOps", "The description for datasource")

	// Empyt datasource (means using system-datasource to provide its data)
	cmd.Flags().BoolVar(&empytDatasource, "empty", false, "Whether to create a empty datasource")

	// Endpoint flags
	cmd.Flags().StringVar(&endpointURL, "endpoint-url", "", "The endpoint url to access datasource.If not provided,a empty datasource will be created")
	if err := cmd.MarkFlagRequired("endpoint-url"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&endpointAuthUser, "endpoint-auth-user", "", "The endpoint's user for datasource authentication")
	cmd.Flags().StringVar(&endpointAuthPwd, "endpoint-auth-password", "", "The endpoint's user password for datasource authentication")
	cmd.Flags().BoolVar(&endpointInsecure, "endpoint-insecure", true, "Whether to access datasource without secure check.Default is yes")

	// Object storage service flags
	cmd.Flags().StringVar(&ossBucket, "oss-bucket", "", "The object storage service bucket name")

	return cmd
}

func DatasourceGetCmd(namespace *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Get datasource",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(os.Args) < 4 {
				return errors.New("missing datasource name")
			}
			name := os.Args[3]

			kubeClient, err := client.GetClient(nil)
			if err != nil {
				return err
			}
			ds, err := datasource.ReadDatasource(cmd.Context(), kubeClient, name, *namespace)
			if err != nil {
				return fmt.Errorf("failed to find datasource: %w", err)
			}

			return printer.PrintYaml(ds)
		},
	}

	return cmd
}

func DatasourceListCmd(namespace *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [usage]",
		Short: "List datasources",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeClient, err := client.GetClient(nil)
			if err != nil {
				return err
			}
			list, err := datasource.ListDatasources(cmd.Context(), kubeClient, generated.ListCommonInput{
				Namespace: *namespace,
			})
			if err != nil {
				return err
			}
			objects := make([]any, len(list.Nodes))
			for i, item := range list.Nodes {
				objects[i] = item
			}
			printer.Print(datasourcePrintHeaders, objects)
			return nil
		},
	}

	return cmd
}

func DatasourceDeleteCmd(namespace *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a datasource",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(os.Args) < 4 {
				return errors.New("missing datasource name")
			}
			name := os.Args[3]

			kubeClient, err := client.GetClient(nil)
			if err != nil {
				return err
			}
			ds, err := datasource.ReadDatasource(cmd.Context(), kubeClient, name, *namespace)
			if err != nil {
				return fmt.Errorf("failed to get datasource: %w", err)
			}
			// delete secrets
			if ds.Endpoint != nil && ds.Endpoint.AuthSecret != nil {
				err = kubeClient.Delete(cmd.Context(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ds.Endpoint.AuthSecret.Name,
						Namespace: *namespace,
					},
				})
				if err != nil {
					return fmt.Errorf("failed to delete auth secret: %w", err)
				}
				klog.Infof("Successfully deleted authsecret %s\n", ds.Endpoint.AuthSecret.Name)
			}
			_, err = datasource.DeleteDatasources(cmd.Context(), kubeClient, &generated.DeleteCommonInput{
				Name:      &name,
				Namespace: *namespace,
			})
			if err != nil {
				return fmt.Errorf("failed to delete datasource: %w", err)
			}
			klog.Infof("Successfully deleted datasource %s\n", name)
			return err
		},
	}

	return cmd
}
