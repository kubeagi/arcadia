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

import "testing"

func TestBytesToSizedStr(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{5242880, "5.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, test := range tests {
		result := BytesToSizedStr(test.bytes)
		if result != test.expected {
			t.Errorf("BytesToSize(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}
