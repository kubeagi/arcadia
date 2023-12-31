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

package utils

import (
	"testing"
	"time"
)

func TestRFC3339Time(t *testing.T) {
	timeStr := "2023-12-01T12:34:56Z"
	expectedTime, _ := time.Parse(time.RFC3339, timeStr)

	// 测试解析正确的时间字符串
	parsedTime, err := RFC3339Time(timeStr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !parsedTime.Equal(expectedTime) {
		t.Errorf("Expected time: %v, but got: %v", expectedTime, parsedTime)
	}

	// 测试解析错误的时间字符串
	invalidTimeStr := "2023-12-01T12:34:56"
	_, err = RFC3339Time(invalidTimeStr)
	if err == nil {
		t.Error("Expected error, but got nil")
	}
}
