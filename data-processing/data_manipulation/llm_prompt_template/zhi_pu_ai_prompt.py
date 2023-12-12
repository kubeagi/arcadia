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
        我会给你一段文本，它们可能包含多个主题内容，学习它们，并整理学习成果，要求为：
        1. 提出最多 25 个问题。
        2. 给出每个问题的答案。
        3. 答案要详细完整，答案可以包含普通文字、链接、代码、表格、公示、媒体链接等 markdown 元素。
        4. 按格式返回多个问题和答案:

        Q1: 问题。
        A1: 答案。
        Q2:
        A2:
        ……

        我的文本：{text}
    """
    
    return prompt_template