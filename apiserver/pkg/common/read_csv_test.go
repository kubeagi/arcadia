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
	"io"
	"reflect"
	"testing"
)

const csvData = `a,a,a
b,b,b
c,c,c
d,d,d
e,e,e
f,f,f
g,g,g
i,i,i
1,1,1
2,2,2
3,3,3
4,4,4
5,5,5
6,6,6`

func TestReadCSV(t *testing.T) {
	type input struct {
		startLine, size, cacheLInes int64

		exp ReadCSVResult
		err error
	}

	reader := bytes.NewReader([]byte(csvData))
	for _, tc := range []input{
		{
			1, 1, 0, ReadCSVResult{[][]string{{"a", "a", "a"}}, 14}, io.EOF,
		},
		{
			1, 2, 14, ReadCSVResult{[][]string{{"a", "a", "a"}, {"b", "b", "b"}}, 14}, nil,
		},
		{
			2, 1, 14, ReadCSVResult{[][]string{{"b", "b", "b"}}, 14}, nil,
		},
		{
			2, 2, 0, ReadCSVResult{[][]string{{"b", "b", "b"}, {"c", "c", "c"}}, 14}, io.EOF,
		},
		{
			9, 10, 14, ReadCSVResult{[][]string{{"1", "1", "1"}, {"2", "2", "2"}, {"3", "3", "3"}, {"4", "4", "4"}, {"5", "5", "5"}, {"6", "6", "6"}}, 14}, io.EOF,
		},
		{
			14, 1, 0, ReadCSVResult{[][]string{{"6", "6", "6"}}, 14}, io.EOF,
		},
		{
			14, 2, 0, ReadCSVResult{[][]string{{"6", "6", "6"}}, 14}, io.EOF,
		},
		{
			8, 3, 14, ReadCSVResult{[][]string{{"i", "i", "i"}, {"1", "1", "1"}, {"2", "2", "2"}}, 14}, nil,
		},
		{
			1, 15, 0, ReadCSVResult{[][]string{
				{"a", "a", "a"},
				{"b", "b", "b"},
				{"c", "c", "c"},
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
				{"g", "g", "g"},
				{"i", "i", "i"},
				{"1", "1", "1"},
				{"2", "2", "2"},
				{"3", "3", "3"},
				{"4", "4", "4"},
				{"5", "5", "5"},
				{"6", "6", "6"},
			}, 14}, io.EOF,
		},
		{
			1, 14, 14, ReadCSVResult{[][]string{
				{"a", "a", "a"},
				{"b", "b", "b"},
				{"c", "c", "c"},
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
				{"g", "g", "g"},
				{"i", "i", "i"},
				{"1", "1", "1"},
				{"2", "2", "2"},
				{"3", "3", "3"},
				{"4", "4", "4"},
				{"5", "5", "5"},
				{"6", "6", "6"},
			}, 14}, io.EOF,
		},
		{
			1, 13, 14, ReadCSVResult{[][]string{
				{"a", "a", "a"},
				{"b", "b", "b"},
				{"c", "c", "c"},
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
				{"g", "g", "g"},
				{"i", "i", "i"},
				{"1", "1", "1"},
				{"2", "2", "2"},
				{"3", "3", "3"},
				{"4", "4", "4"},
				{"5", "5", "5"},
			}, 14}, nil,
		},

		{
			15, 2, 0, ReadCSVResult{nil, 14}, io.EOF,
		},
		{
			// page=1, size=3
			1, 3, 0, ReadCSVResult{[][]string{
				{"a", "a", "a"},
				{"b", "b", "b"},
				{"c", "c", "c"},
			}, 14}, io.EOF,
		},
		{
			// page=2,size=3
			4, 3, 14, ReadCSVResult{[][]string{
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
			}, 14}, nil,
		},
		{
			// page=3,size=3
			7, 3, 14, ReadCSVResult{[][]string{
				{"g", "g", "g"},
				{"i", "i", "i"},
				{"1", "1", "1"},
			}, 14}, nil,
		},
		{
			// page=4,size=3
			10, 3, 14, ReadCSVResult{[][]string{
				{"2", "2", "2"},
				{"3", "3", "3"},
				{"4", "4", "4"},
			}, 14}, nil,
		},
		{
			// page=5,size=3
			13, 3, 14, ReadCSVResult{[][]string{
				{"5", "5", "5"},
				{"6", "6", "6"},
			}, 14}, io.EOF,
		},
		{
			13, 3, 0, ReadCSVResult{[][]string{
				{"5", "5", "5"},
				{"6", "6", "6"},
			}, 14}, io.EOF,
		},
		{
			14, 1, 0, ReadCSVResult{[][]string{
				{"6", "6", "6"},
			}, 14}, io.EOF,
		},
		{
			4, 5, 14, ReadCSVResult{[][]string{
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
				{"g", "g", "g"},
				{"i", "i", "i"},
			}, 14}, nil,
		},
	} {
		if r, err := ReadCSV(reader, tc.startLine, tc.size, tc.cacheLInes); err != tc.err || !reflect.DeepEqual(tc.exp, r) {
			t.Fatalf("with input %+v expect %+v get %+v error %s", tc, tc.exp, r, err)
		}
		_, _ = reader.Seek(0, io.SeekStart)
	}
}
