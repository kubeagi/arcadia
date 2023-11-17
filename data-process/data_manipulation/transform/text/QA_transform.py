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

import re

import zhipuai
from common import config

###
# QA生成
# @author: wangxinbiao
# @date: 2023-11-15 16:30:01
# modify history
# ==== 2023-11-15 16:30:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def generate_QA(opt={}):
    zhipuai.api_key = config.zhipuai_api_key

    text = opt['text']
    content = """
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

        我的文本：
    """

    content = content + text

    response = zhipuai.model_api.invoke(
        model="chatglm_6b",
        prompt=[{"role": "user", "content": content}],
        top_p=0.7,
        temperature=0.9,
    )

    # 格式化后的QA对
    result = await formatSplitText(response['data']['choices'][0]['content'])

    return result


###
# 对QA进行格式化
# @author: wangxinbiao
# @date: 2023-11-15 16:30:01
# modify history
# ==== 2023-11-15 16:30:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def formatSplitText(text):

    pattern = re.compile(r'Q\d+:(\s*)(.*?)(\s*)A\d+:(\s*)([\s\S]*?)(?=Q|$)')

    # 移除换行符
    text = text.replace('\\n', '')
    matches = pattern.findall(text)

    result = []
    for match in matches:
        q = match[1]
        a = match[4]
        if q and a:
            result.append({
                'q': q,
                'a': a
            })

    return result
