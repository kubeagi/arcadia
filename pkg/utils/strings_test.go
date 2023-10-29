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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddString(t *testing.T) {
	finalizers := []string{"a", "b"}
	f := "c"

	result := AddString(finalizers, f)

	// Assert that the element "c" is added to the finalizers slice
	expected := []string{"a", "b", "c"}
	assert.Equal(t, expected, result)
}

func TestRemoveString(t *testing.T) {
	finalizers := []string{"a", "b", "c"}
	f := "b"

	result := RemoveString(finalizers, f)

	// Assert that the element "b" is removed from the finalizers slice
	expected := []string{"a", "c"}
	assert.Equal(t, expected, result)
}

func TestContainString(t *testing.T) {
	finalizers := []string{"a", "b", "c"}
	f := "b"

	result := ContainString(finalizers, f)

	// Assert that the element "b" is present in the finalizers slice
	assert.True(t, result)
}

func TestAddOrSwapString(t *testing.T) {
	strings := []string{"a", "b", "c"}
	str := "b"

	changed, result := AddOrSwapString(strings, str)

	// Assert that the element "b" is swapped with the last element "c"
	expected := []string{"a", "c", "b"}
	assert.True(t, changed)
	assert.Equal(t, expected, result)
}
