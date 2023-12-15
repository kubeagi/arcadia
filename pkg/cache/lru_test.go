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
package cache

import (
	"container/list"
	"reflect"
	"sync"
	"testing"
)

func limitGreaterThanZero(t *testing.T) {
	_, err := NewLRU(0)
	if err == nil {
		t.Fatalf("limit is 0, should return error")
	}
}

func toIntArray(l *lru) []int {
	result := make([]int, 0)
	for e := l.list.Front(); e != nil; e = e.Next() {
		item := e.Value.(lruItem)
		result = append(result, item.val.(int))
	}
	return result
}

type lruInput struct {
	action    string // get, set
	key, val  int
	expResult []int
}

func TestLRUCache(t *testing.T) {
	limitGreaterThanZero(t)

	// The direct use of structures here is for printing purposes to ensure that the data is correct.
	lc := &lru{limit: 5, m: sync.Mutex{}, cache: make(map[any]*list.Element), list: list.New()}
	testCases := []lruInput{
		{action: "set", key: 1, val: 1, expResult: []int{1}},
		{action: "set", key: 2, val: 2, expResult: []int{2, 1}},
		{action: "set", key: 3, val: 3, expResult: []int{3, 2, 1}},
		{action: "set", key: 4, val: 4, expResult: []int{4, 3, 2, 1}},
		{action: "set", key: 5, val: 5, expResult: []int{5, 4, 3, 2, 1}},
		// get
		{action: "get", key: 3, val: 3, expResult: []int{3, 5, 4, 2, 1}},
		{action: "get", key: 1, val: 1, expResult: []int{1, 3, 5, 4, 2}},
		{action: "get", key: 4, val: 4, expResult: []int{4, 1, 3, 5, 2}},
		{action: "get", key: 2, val: 2, expResult: []int{2, 4, 1, 3, 5}},
		// update value
		{action: "set", key: 4, val: 100, expResult: []int{100, 2, 1, 3, 5}},

		// push others
		{action: "set", key: 10, val: 10, expResult: []int{10, 100, 2, 1, 3}},
		{action: "set", key: 11, val: 11, expResult: []int{11, 10, 100, 2, 1}},
		{action: "set", key: 10, val: 10, expResult: []int{10, 11, 100, 2, 1}},
		{action: "set", key: 5, val: 5, expResult: []int{5, 10, 11, 100, 2}},
		{action: "set", key: 1, val: 1, expResult: []int{1, 5, 10, 11, 100}},

		// get
		{action: "get", key: 1, val: 1, expResult: []int{1, 5, 10, 11, 100}},
		{action: "get", key: 4, val: 100, expResult: []int{100, 1, 5, 10, 11}},

		// delete
		{action: "del", key: 4, expResult: []int{1, 5, 10, 11}},
		{action: "del", key: 1, expResult: []int{5, 10, 11}},
		{action: "del", key: 5, expResult: []int{10, 11}},
		{action: "del", key: 10, expResult: []int{11}},
		{action: "del", key: 11, expResult: []int{}},

		// again
		{action: "set", key: 1, val: 1, expResult: []int{1}},
		{action: "set", key: 2, val: 2, expResult: []int{2, 1}},
		{action: "set", key: 3, val: 3, expResult: []int{3, 2, 1}},
		{action: "set", key: 4, val: 4, expResult: []int{4, 3, 2, 1}},
		{action: "set", key: 5, val: 5, expResult: []int{5, 4, 3, 2, 1}},
		// get
		{action: "get", key: 3, val: 3, expResult: []int{3, 5, 4, 2, 1}},
		{action: "get", key: 1, val: 1, expResult: []int{1, 3, 5, 4, 2}},
		{action: "get", key: 4, val: 4, expResult: []int{4, 1, 3, 5, 2}},
		{action: "get", key: 2, val: 2, expResult: []int{2, 4, 1, 3, 5}},
		// update value
		{action: "set", key: 4, val: 100, expResult: []int{100, 2, 1, 3, 5}},

		// push others
		{action: "set", key: 10, val: 10, expResult: []int{10, 100, 2, 1, 3}},
		{action: "set", key: 11, val: 11, expResult: []int{11, 10, 100, 2, 1}},
		{action: "set", key: 10, val: 10, expResult: []int{10, 11, 100, 2, 1}},
		{action: "set", key: 5, val: 5, expResult: []int{5, 10, 11, 100, 2}},
		{action: "set", key: 1, val: 1, expResult: []int{1, 5, 10, 11, 100}},

		// get
		{action: "get", key: 1, val: 1, expResult: []int{1, 5, 10, 11, 100}},
		{action: "get", key: 4, val: 100, expResult: []int{100, 1, 5, 10, 11}},
	}
	for _, tc := range testCases {
		if tc.action == "set" {
			_ = lc.Set(tc.key, tc.val)
		}
		if tc.action == "get" {
			r1, _ := lc.Get(tc.key)
			if r1.(int) != tc.val {
				t.Fatalf("with input %v, expect value %v, get value %v", tc, tc.val, r1)
			}
		}
		if tc.action == "del" {
			_ = lc.Delete(tc.key)
		}
		r := toIntArray(lc)
		if !reflect.DeepEqual(r, tc.expResult) {
			t.Fatalf("with input %v, expect %v get %v", tc, tc.expResult, r)
		}
	}
}
