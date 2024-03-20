/*
Copyright 2024 KubeAGI.

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
package forwardrepo

import (
	"context"
	"reflect"
	"testing"
)

const (
	README  = "\n# Qwen1.5-0.5B\n\n\n## Introduction\n\nQwen1.5 is the beta version of Qwen2, a transformer-based decoder-only language model pretrained on a large amount of data. In comparison with the previous released Qwen, the improvements include:\n\n* 6 model sizes, including 0.5B, 1.8B, 4B, 7B, 14B, and 72B;\n* Significant performance improvement in Chat models;\n* Multilingual support of both base and chat models;\n* Stable support of 32K context length for models of all sizes\n* No need of `trust_remote_code`.\n\nFor more details, please refer to our [blog post](https://qwenlm.github.io/blog/qwen1.5/) and [GitHub repo](https://github.com/QwenLM/Qwen1.5).\n\n\n## Model Details\nQwen1.5 is a language model series including decoder language models of different model sizes. For each size, we release the base language model and the aligned chat model. It is based on the Transformer architecture with SwiGLU activation, attention QKV bias, group query attention, mixture of sliding window attention and full attention, etc. Additionally, we have an improved tokenizer adaptive to multiple natural languages and codes. For the beta version, temporarily we did not include GQA and the mixture of SWA and full attention.\n\n## Requirements\nThe code of Qwen1.5 has been in the latest Hugging face transformers and we advise you to install `transformers\u003e=4.37.0`, or you might encounter the following error:\n```\nKeyError: 'qwen2'.\n```\n\n\n## Usage\n\nWe do not advise you to use base language models for text generation. Instead, you can apply post-training, e.g., SFT, RLHF, continued pretraining, etc., on this model.\n\n\n## Citation\n\nIf you find our work helpful, feel free to give us a cite.\n\n```\n@article{qwen,\n  title={Qwen Technical Report},\n  author={Jinze Bai and Shuai Bai and Yunfei Chu and Zeyu Cui and Kai Dang and Xiaodong Deng and Yang Fan and Wenbin Ge and Yu Han and Fei Huang and Binyuan Hui and Luo Ji and Mei Li and Junyang Lin and Runji Lin and Dayiheng Liu and Gao Liu and Chengqiang Lu and Keming Lu and Jianxin Ma and Rui Men and Xingzhang Ren and Xuancheng Ren and Chuanqi Tan and Sinan Tan and Jianhong Tu and Peng Wang and Shijie Wang and Wei Wang and Shengguang Wu and Benfeng Xu and Jin Xu and An Yang and Hao Yang and Jian Yang and Shusheng Yang and Yang Yao and Bowen Yu and Hongyi Yuan and Zheng Yuan and Jianwei Zhang and Xingxuan Zhang and Yichang Zhang and Zhenru Zhang and Chang Zhou and Jingren Zhou and Xiaohuan Zhou and Tianhang Zhu},\n  journal={arXiv preprint arXiv:2309.16609},\n  year={2023}\n}\n```\n"
	modelID = "qwen/Qwen1.5-0.5B"
)

var (
	Revisions = Revision{
		Branches: []BranchTag{
			{Name: "master"},
		},
	}
)

func TestModelScope(t *testing.T) {
	m := NewModelScope(WithModelID(modelID))
	ctx := context.Background()
	r, err := m.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if r != README {
		t.Fatalf("expect %s get %s", README, r)
	}

	revisions, err := m.Revisions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(revisions, Revisions) {
		t.Fatalf("expect %v get %v", Revisions, revisions)
	}
}
