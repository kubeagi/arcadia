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
	"sync"
)

type memory struct {
	s sync.Map
}

// globally unique
var m *memory

func NewMemCache() Cache {
	if m == nil {
		m = &memory{s: sync.Map{}}
	}
	return m
}

func (m *memory) Set(key any, val any) error {
	m.s.Store(key, val)
	return nil
}

func (m *memory) Get(key any) (v any, ok bool) {
	return m.s.Load(key)
}

func (m *memory) Delete(key any) error {
	m.s.Delete(key)
	return nil
}
