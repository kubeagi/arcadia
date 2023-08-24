package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const (
	RBACInquiryTemplate = `
I have an RBAC configuration in my Kubernetes cluster that I'm interested in assessing the security implications of this RBAC configuration.


Below is a snippet of RBAC configuration:

  {{.Context}}


**Answer in {{.Language}}**

**RBAC Assessment**:
   - Could you please review this RBAC configuration and inform me of any security risks, permissions, or potential vulnerabilities that might be associated with it?

**Final Rating(Must given)**:
   - On a scale of 1 to 10, with 10 being the most secure, how would you rate the security of this RBAC configuration?

I appreciate your assistance in evaluating this RBAC configuration for security purposes. 
`
)

type InquiryData struct {
	Context  string
	Language string
}

var (
	rbacFile string
	language string
)

func Inquiry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inquiry [args]",
		Short: "RBAC inquiry with AI",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := zhipuai.NewZhiPuAI(apiKey)

			// read rbac content from file
			rbacContent, err := os.ReadFile(rbacFile)
			if err != nil {
				return err
			}
			// Prepare data for template
			data := InquiryData{
				Context:  strings.TrimSpace(string(rbacContent)),
				Language: language,
			}

			// Create a new template and parse the RBAC inquiry template
			tmpl := template.Must(template.New("RBACInquiry").Parse(RBACInquiryTemplate))

			// Execute the template with the data
			var output strings.Builder
			if err := tmpl.Execute(&output, data); err != nil {
				log.Fatal(err)
			}

			params := zhipuai.DefaultModelParams()
			params.Model = zhipuai.Model(model)
			params.Method = zhipuai.Method(method)
			params.Prompt = []zhipuai.Prompt{
				{Role: zhipuai.User, Content: output.String()},
			}

			klog.V(0).Infoln("RBAC Inquiry with the help of AI")
			if params.Method == zhipuai.ZhiPuAIInvoke {
				resp, err := client.Invoke(params)
				if err != nil {
					return err
				}
				if resp.Code != 200 {
					return fmt.Errorf("inquiry failed: %s", resp.String())
				}
				klog.V(0).Info(resp.Data.Choices[0].Content)
				return nil
			}

			if err = client.SSEInvoke(params, nil); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&rbacFile, "file", "f", "", "rbac file to be inquired")
	cmd.Flags().StringVarP(&language, "language", "l", "English", "Language in the response")

	cmd.MarkFlagRequired("file")

	return cmd
}
