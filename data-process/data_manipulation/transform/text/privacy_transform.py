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
# 去隐私
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
# 去除邮箱地址
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def remove_email(opt={}):
    text = opt['text']
    
    try:
        email_pattern = r'[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'

        # 将邮箱地址替换为 "PI:EMAIL"
        replacement_text = "PI:EMAIL"

        cleaned_text = re.sub(email_pattern, replacement_text, chinese_text)
        return clean_text

    except Exception as ex:
        return ''
    