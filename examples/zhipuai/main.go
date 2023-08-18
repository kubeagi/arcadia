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

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 0 {
		panic("api key is empty")
	}
	apiKey := os.Args[1]
	resp, err := sampleInvoke(apiKey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("SampleInvoke: \n %+v\n", resp)

	resp, err = sampleInvokeAsync(apiKey)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("sampleInvokeAsync: \n %+v\n", resp)
	// taskID := "76997570932704279317856632766629711813"
	// resp, err = getInvokeAsyncResult(apiKey, taskID)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("getInvokeAsyncResult: \n %+v\n", resp)
}
