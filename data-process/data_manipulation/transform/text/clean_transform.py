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


###
# 异常清洗
# @author: wangxinbiao
# @date: 2023-11-01 10:44:01
# modify history
# ==== 2023-11-01 10:44:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

import re

###
# 去除不可见字符
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def remove_invisible_characters(opt={}):
    text = opt['text']
    
    try:
        clean_text = re.sub(r'[\x00-\x1F\x7F-\x9F\xAD\r\n\t\b\x0B\x1C\x1D\x1E]', '', text)
        return {
            'status': 200,
            'message': '',
            'data': clean_text
        }

    except Exception as ex:
        return {
            'status': 400,
            'message': '去除不可见字符失败：' + str(ex),
            'data': ''
        }
