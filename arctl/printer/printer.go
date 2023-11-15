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

package printer

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

func Print(headers []string, objs []any) {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 4, ' ', 0)
	headersCopy := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		headersCopy[i] = strings.ToUpper(headers[i])
	}

	fmt.Fprintln(w, strings.Join(headersCopy, "\t"))
	row := make([]string, len(headers))
	for _, o := range objs {
		for i := 0; i < len(headers); i++ {
			row[i] = GetByHeader(o, headers[i])
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func GetByHeader(obj any, s string) string {
	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if strings.TrimSuffix(field.Tag.Get("json"), ",omitempty") == s {
			fieldValue := v.Field(i)
			if fieldValue.Kind() == reflect.Struct || fieldValue.Kind() == reflect.Ptr {
				jsonBytes, err := json.Marshal(fieldValue.Interface())
				if err != nil {
					return fmt.Sprintf("Error marshaling JSON: %v", err)
				}
				return string(jsonBytes)
			}
			return fmt.Sprintf("%v", fieldValue)
		}
	}

	return "<none>"
}
