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
	"reflect"
	"testing"
)

var (
	nilany map[string]interface{}
	nilstr map[string]string
)

func TestMapAny2Str(t *testing.T) {
	type args struct {
		input map[string]interface{}
	}

	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "common test",
			args: args{
				input: map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "empty",
			args: args{
				input: map[string]interface{}{},
			},
			want: map[string]string{},
		},
		{
			name: "nil",
			args: args{
				input: nilany,
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapAny2Str(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapAny2Str() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapStr2Any(t *testing.T) {
	type args struct {
		input map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]any
	}{
		{
			name: "common test",
			args: args{
				input: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			want: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "empty",
			args: args{
				input: map[string]string{},
			},
			want: map[string]any{},
		},
		{
			name: "nil",
			args: args{
				input: nilstr,
			},
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapStr2Any(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapStr2Any() = %v, want %v", got, tt.want)
			}
		})
	}
}
