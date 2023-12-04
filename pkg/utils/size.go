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

package utils

import "fmt"

const (
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
)

// BytesToSize converts size(byte as the uit) to string with appropriate unit ending
func BytesToSizedStr(bytes int64) string {
	precision := 2

	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < MB {
		kilobytes := float64(bytes) / KB
		return fmt.Sprintf("%.*f KB", precision, kilobytes)
	}
	if bytes < GB {
		megabytes := float64(bytes) / MB
		precision := 2
		return fmt.Sprintf("%.*f MB", precision, megabytes)
	}

	megabytes := float64(bytes) / GB
	return fmt.Sprintf("%.*f GB", precision, megabytes)
}
