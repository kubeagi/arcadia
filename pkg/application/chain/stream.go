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

package chain

import (
	"context"
	"errors"

	"k8s.io/klog/v2"
)

func stream(res map[string]any) func(ctx context.Context, chunk []byte) error {
	return func(ctx context.Context, chunk []byte) error {
		if _, ok := res["answer_stream"]; !ok {
			res["answer_stream"] = make(chan string)
		}
		streamChan, ok := res["answer_stream"].(chan string)
		if !ok {
			klog.Errorln("answer_stream is not chan string")
			return errors.New("answer_stream is not chan string")
		}
		klog.V(5).Infoln("stream out:", string(chunk))
		streamChan <- string(chunk)
		return nil
	}
}
