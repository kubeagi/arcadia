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
	"bytes"
	"encoding/csv"
	"io"
	"sort"
)

type (
	ReadCSVResult struct {
		Rows    [][]string `json:"rows"`
		Total   int64      `json:"total"`
		Version string     `json:"version"`
	}

	CSVLine struct {
		LinNumber uint64   `json:"lineNumber,omitempty"`
		Values    []string `json:"values,omitempty"`
	}
	UpdateCSVBody struct {
		// BucketName string `json:"bucketName"`, from header
		BucketPath string `json:"bucketPath"`
		FileName   string `json:"fileName"`
		Version    string `json:"version"`

		// The version to be edited is inconsistent with the latest version.
		// Do you want to force an update?
		ForceUpdate bool `json:"forceUpdate"`

		UpdateLines []CSVLine  `json:"updateLines"`
		NewLines    [][]string `json:"newLines,omitempty"`
		DelLines    []int      `json:"delLines"`
	}
)

// ReadCSV Reads the contents of a csv file by lines according to startLine and lines,
// if cacheLines=0 then the total number of lines in the file will be counted,
// otherwise the total number of lines in the file will not be counted and the speedup function returns.
func ReadCSV(o io.Reader, startLine, lines, cachedTotalLines int64) (ReadCSVResult, error) {
	result := ReadCSVResult{}
	var (
		line []string
		cur  int64 = 0
		err  error
	)
	csvReader := csv.NewReader(o)

	for {
		line, err = csvReader.Read()
		if err != nil {
			if err != io.EOF {
				result.Rows = nil
				result.Total = 0
			}
			break
		}

		result.Total++
		cur++

		if cur < startLine {
			continue
		}

		// If the target number of rows has already been read,
		// determine whether to continue execution or jump out of the loop based on whether all rows have been read.
		if cur >= startLine+lines {
			if cachedTotalLines != 0 {
				break
			}
			continue
		}

		result.Rows = append(result.Rows, line)
	}

	if cachedTotalLines != 0 {
		result.Total = cachedTotalLines
	}
	return result, err
}

func EditCSV(r io.Reader, updateLines []CSVLine, addLines [][]string, delLines []int) (io.Reader, int64, error) {
	reader := csv.NewReader(r)

	sort.Slice(updateLines, func(i, j int) bool {
		return updateLines[i].LinNumber < updateLines[j].LinNumber
	})
	sort.Ints(delLines)

	var (
		line   []string
		err    error
		ui, di int
		write  bool
	)

	buf := bytes.NewBuffer([]byte{})
	w := csv.NewWriter(buf)

	li := 0
	for ; ; li++ {
		line, err = reader.Read()
		if err != nil {
			if err != io.EOF {
				return nil, 0, err
			}
			break
		}
		if li == 0 {
			// write header
			if err = w.Write(line); err != nil {
				return nil, 0, err
			}
			for _, newLine := range addLines {
				if err = w.Write(newLine); err != nil {
					return nil, 0, err
				}
			}
			continue
		}

		write = true
		if di < len(delLines) && li == delLines[di] {
			write = false
			di++
		}

		if ui < len(updateLines) && updateLines[ui].LinNumber == uint64(li) {
			line = updateLines[ui].Values
			ui++
		}
		if !write {
			continue
		}
		if err = w.Write(line); err != nil {
			return nil, 0, err
		}
	}
	w.Flush()
	return buf, int64(len(buf.Bytes())), nil
}
