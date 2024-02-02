# Copyright 2023 KubeAGI.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


def get_default_prompt_template():
    prompt_template = """
    {text}

    请将上述内容按照问答的方式，提出不超过 25 个问题，并给出每个问题的答案，每个问题必须有 Q 和对应的 A，并严格按照以下方式展示： Q1: 问题。\n  A1: 答案。\n Q2: 问题 \n A2: 答案\n  注意，尽可能多的提出问题，但是 Q 不要重复，也不要出现只有 Q 没有 A 的情况。
    """

    return prompt_template
