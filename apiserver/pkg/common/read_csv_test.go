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
			1, 1, 0, ReadCSVResult{[][]string{{"a", "a", "a"}}, 14, ""}, io.EOF,
		},
		{
			1, 2, 14, ReadCSVResult{[][]string{{"a", "a", "a"}, {"b", "b", "b"}}, 14, ""}, nil,
		},
		{
			2, 1, 14, ReadCSVResult{[][]string{{"b", "b", "b"}}, 14, ""}, nil,
		},
		{
			2, 2, 0, ReadCSVResult{[][]string{{"b", "b", "b"}, {"c", "c", "c"}}, 14, ""}, io.EOF,
		},
		{
			9, 10, 14, ReadCSVResult{[][]string{{"1", "1", "1"}, {"2", "2", "2"}, {"3", "3", "3"}, {"4", "4", "4"}, {"5", "5", "5"}, {"6", "6", "6"}}, 14, ""}, io.EOF,
		},
		{
			14, 1, 0, ReadCSVResult{[][]string{{"6", "6", "6"}}, 14, ""}, io.EOF,
		},
		{
			14, 2, 0, ReadCSVResult{[][]string{{"6", "6", "6"}}, 14, ""}, io.EOF,
		},
		{
			8, 3, 14, ReadCSVResult{[][]string{{"i", "i", "i"}, {"1", "1", "1"}, {"2", "2", "2"}}, 14, ""}, nil,
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
			}, 14, ""}, io.EOF,
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
			}, 14, ""}, io.EOF,
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
			}, 14, ""}, nil,
		},

		{
			15, 2, 0, ReadCSVResult{nil, 14, ""}, io.EOF,
		},
		{
			// page=1, size=3
			1, 3, 0, ReadCSVResult{[][]string{
				{"a", "a", "a"},
				{"b", "b", "b"},
				{"c", "c", "c"},
			}, 14, ""}, io.EOF,
		},
		{
			// page=2,size=3
			4, 3, 14, ReadCSVResult{[][]string{
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
			}, 14, ""}, nil,
		},
		{
			// page=3,size=3
			7, 3, 14, ReadCSVResult{[][]string{
				{"g", "g", "g"},
				{"i", "i", "i"},
				{"1", "1", "1"},
			}, 14, ""}, nil,
		},
		{
			// page=4,size=3
			10, 3, 14, ReadCSVResult{[][]string{
				{"2", "2", "2"},
				{"3", "3", "3"},
				{"4", "4", "4"},
			}, 14, ""}, nil,
		},
		{
			// page=5,size=3
			13, 3, 14, ReadCSVResult{[][]string{
				{"5", "5", "5"},
				{"6", "6", "6"},
			}, 14, ""}, io.EOF,
		},
		{
			13, 3, 0, ReadCSVResult{[][]string{
				{"5", "5", "5"},
				{"6", "6", "6"},
			}, 14, ""}, io.EOF,
		},
		{
			14, 1, 0, ReadCSVResult{[][]string{
				{"6", "6", "6"},
			}, 14, ""}, io.EOF,
		},
		{
			4, 5, 14, ReadCSVResult{[][]string{
				{"d", "d", "d"},
				{"e", "e", "e"},
				{"f", "f", "f"},
				{"g", "g", "g"},
				{"i", "i", "i"},
			}, 14, ""}, nil,
		},
	} {
		if r, err := ReadCSV(reader, tc.startLine, tc.size, tc.cacheLInes); err != tc.err || !reflect.DeepEqual(tc.exp, r) {
			t.Fatalf("with input %+v expect %+v get %+v error %s", tc, tc.exp, r, err)
		}
		_, _ = reader.Seek(0, io.SeekStart)
	}
}

const (
	editCsv = `hc1,hc2
a,a
b,b
c,c
d,d
`
)

func TestEditCSV(t *testing.T) {
	type input struct {
		body     UpdateCSVBody
		expbytes int64
		exp      string
	}
	for _, tc := range []input{
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}}},
			expbytes: int64(len(editCsv)),
			exp: `hc1,hc2
A,A
b,b
c,c
d,d
`,
		},

		{body: UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}, {LinNumber: 2, Values: []string{"B", "B"}}}},
			expbytes: int64(len(editCsv)),
			exp: `hc1,hc2
A,A
B,B
c,c
d,d
`,
		},
		{body: UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}, {LinNumber: 2, Values: []string{"B", "B"}}, {LinNumber: 3, Values: []string{"C", "C"}}}},
			expbytes: int64(len(editCsv)),
			exp: `hc1,hc2
A,A
B,B
C,C
d,d
`,
		},
		{body: UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}, {LinNumber: 2, Values: []string{"B", "B"}}, {LinNumber: 3, Values: []string{"C", "C"}}, {LinNumber: 4, Values: []string{"D", "D"}}}},
			expbytes: int64(len(editCsv)),
			exp: `hc1,hc2
A,A
B,B
C,C
D,D
`},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}}, DelLines: []int{1, 2, 3, 4}},
			expbytes: int64(8),
			exp: `hc1,hc2
`,
		},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}}, DelLines: []int{1, 2, 3}},
			expbytes: int64(12),
			exp: `hc1,hc2
d,d
`,
		},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}}, DelLines: []int{1, 2}},
			expbytes: int64(16),
			exp: `hc1,hc2
c,c
d,d
`,
		},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}}, DelLines: []int{1}},
			expbytes: int64(20),
			exp: `hc1,hc2
b,b
c,c
d,d
`,
		},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 1, Values: []string{"A", "A"}}}, DelLines: []int{1}},
			expbytes: int64(20),
			exp: `hc1,hc2
b,b
c,c
d,d
`,
		},
		{
			body:     UpdateCSVBody{NewLines: [][]string{{"line1", "value1"}, {"line2", "value2"}}, DelLines: []int{1}},
			expbytes: int64(46),
			exp: `hc1,hc2
line1,value1
line2,value2
b,b
c,c
d,d
`,
		},
		{
			body:     UpdateCSVBody{NewLines: [][]string{{"line1", "value1"}, {"line2", "value2"}}, DelLines: []int{1, 2}},
			expbytes: int64(42),
			exp: `hc1,hc2
line1,value1
line2,value2
c,c
d,d
`,
		},
		{
			body:     UpdateCSVBody{NewLines: [][]string{{"line1", "value1"}, {"line2", "value2"}}, DelLines: []int{1, 2, 3}},
			expbytes: int64(38),
			exp: `hc1,hc2
line1,value1
line2,value2
d,d
`,
		},
		{
			body:     UpdateCSVBody{NewLines: [][]string{{"line1", "value1"}, {"line2", "value2"}}, DelLines: []int{1, 2, 3, 4}},
			expbytes: int64(34),
			exp: `hc1,hc2
line1,value1
line2,value2
`,
		},
		{
			body:     UpdateCSVBody{NewLines: [][]string{{"line1", "value1"}, {"line2", "value2"}}, DelLines: []int{1, 2, 3, 4}},
			expbytes: int64(34),
			exp: `hc1,hc2
line1,value1
line2,value2
`,
		},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 4, Values: []string{"e", "e"}}}, NewLines: [][]string{{"line1", "value1"}}, DelLines: []int{1, 2, 3}},
			expbytes: 25,
			exp: `hc1,hc2
line1,value1
e,e
`,
		},
		{
			body: UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 4, Values: []string{"e", "e"}}, {LinNumber: 3, Values: []string{"f", "f"}}},
				NewLines: [][]string{{"line1", "value1"}}, DelLines: []int{1, 2}},
			expbytes: 29,
			exp: `hc1,hc2
line1,value1
f,f
e,e
`,
		},
		{
			body: UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 4, Values: []string{"e", "e"}}, {LinNumber: 2, Values: []string{"f", "f"}}, {LinNumber: 3, Values: []string{"i", "i"}}},
				NewLines: [][]string{{"line1", "value1"}}, DelLines: []int{1}},
			expbytes: 33,
			exp: `hc1,hc2
line1,value1
f,f
i,i
e,e
`,
		},
		{
			body:     UpdateCSVBody{UpdateLines: []CSVLine{{LinNumber: 4, Values: []string{"e", "e"}}}, NewLines: [][]string{{"line1", "value1"}}},
			expbytes: 37,
			exp: `hc1,hc2
line1,value1
a,a
b,b
c,c
e,e
`,
		},
	} {
		r := bytes.NewBufferString(editCsv)
		result, l, _ := EditCSV(r, tc.body.UpdateLines, tc.body.NewLines, tc.body.DelLines)
		if l != tc.expbytes {
			t.Fatalf("expect %d bytes, actual are %d bytes", tc.expbytes, l)
		}

		data, _ := io.ReadAll(result)
		if string(data) != tc.exp {
			t.Fatalf("expect %v get %v", tc.exp, data)
		}
	}
}
