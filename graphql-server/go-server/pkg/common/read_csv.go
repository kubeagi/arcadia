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
package common

import (
	"encoding/csv"
	"io"
)

// ReadCSV function reads the data in lines from startLine and returns an error if there is an error,
// you can determine if there is still data by determining if err is io.
func ReadCSV(o io.Reader, startLine, lines int64) ([][]string, error) {
	var (
		line        []string
		err         error
		cur         = int64(0)
		recordLines = int64(0)
		result      [][]string
	)

	csvReader := csv.NewReader(o)

	for {
		line, err = csvReader.Read()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		cur++
		if cur >= startLine {
			if recordLines >= lines {
				break
			}
			result = append(result, line)
			recordLines++
		}
	}

	return result, err
}
