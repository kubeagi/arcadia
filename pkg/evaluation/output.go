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

package evaluation

import (
	"encoding/csv"
	"fmt"
	"strings"
)

type Output interface {
	Output(RagasDataRow) error
}

// PrintOutput
type PrintOutput struct{}

// Output this row to standard output
func (print *PrintOutput) Output(row RagasDataRow) error {
	fmt.Printf("question:%s \n ground_truths:%s \n answer:%s \n contexts:%v \n latency:%s \n", row.Question, row.GroundTruths, row.Answer, row.Contexts, row.Latency)
	return nil
}

// CSVOutput writes row to csv
type CSVOutput struct {
	W *csv.Writer
}

// Output a row to csv
func (csv *CSVOutput) Output(row RagasDataRow) error {
	return csv.W.Write([]string{row.Question, strings.Join(row.GroundTruths, ";"), row.Answer, strings.Join(row.Contexts, ";"), row.Latency})
}
