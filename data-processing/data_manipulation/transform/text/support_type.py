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
# 数据处理支持的类型
# @author: wangxinbiao
# @date: 2023-11-02 14:25:01
# modify history
# ==== 2023-11-02 14:25:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

support_types = [
    {
        'name': 'clean',
        'description': '异常清洗',
        'children': [
            {
                'name': 'remove_invisible_characters',
                'zh_name': '移除不可见字符',
                'description': '移除ASCII中的一些不可见字符, 如0-32 和127-160这两个范围'
            },
            {
                'name': 'space_standardization',
                'zh_name': '规范化空格',
                'description': '将不同的unicode空格比如u2008, 转成正常的空格'
            },
            {
                'name': 'remove_garbled_text',
                'zh_name': '去除乱码',
                'description': '去除乱码和无意义的unicode'
            },
            {
                'name': 'traditional_to_simplified',
                'zh_name': '繁体转简体',
                'description': '繁体转简体，如“不經意，妳的笑容”清洗成“不经意，你的笑容”'
            },
            {
                'name': 'remove_html_tag',
                'zh_name': '去除网页标识符',
                'description': '移除文档中的html标签, 如<html>,<dev>,<p>等'
            },
            {
                'name': 'remove_emojis',
                'zh_name': '去除表情',
                'description': '去除文档中的表情，如‘🐰’, ‘🧑🏼’等'
            }
        ]
    },
    {
        'name': 'filtration',
        'description': '过滤',
        'children': [
            {
                'name': 'word_count',
                'zh_name': '检查文档的词数目',
                'description': '词数目不在指定范围会被过滤掉，如中文[1,1000000]'
            },
            {
                'name': 'character_duplication_rate',
                'zh_name': '检查文档的字重复率',
                'description': '如果字重复率太高，意味着文档中重复的字太多，文档会被过滤掉'
            },
            {
                'name': 'word_duplication_rate',
                'zh_name': '检查文档的词重复率',
                'description': '如果词重复率太高，意味着文档中重复的词太多，文档会被过滤掉'
            },
            {
                'name': 'special_character_rate',
                'zh_name': '检查文档的特殊字符率',
                'description': '如果特殊字符率太高，意味着文档中特殊字符太多，文档会被过滤掉'
            },
            {
                'name': 'pornography_violence_word_rate',
                'zh_name': '检查文档的色情暴力词率',
                'description': '如果色情暴力词率太高，文档会被过滤掉'
            },
            {
                'name': 'language_probability',
                'zh_name': '检查文档的语言概率',
                'description': '如果语言概率太低，文档会被过滤掉'
            },
            {
                'name': 'perplexity',
                'zh_name': '检查文档的困惑度',
                'description': '如果困惑度太高，文档会被过滤掉'
            }
        ]
    },
    {
        'name': 'duplicates',
        'description': '去重',
        'children': [
            {
                'name': 'simhash_operator',
                'zh_name': 'simhash-operator',
                'description': '根据海明距离计算文档相似度, 相似度<=海明距离，认为两个文档相似。（范围：4-6）'
            }
        ]
    },
    {
        'name': 'privacy_erosion',
        'description': '去隐私',
        'children': [
            {
                'name': 'remove_email',
                'zh_name': '去除邮箱',
                'description': '去除email地址'
            },
            {
                'name': 'remove_ip_address',
                'zh_name': '去除IP地址',
                'description': '去除IPv4 或者 IPv6 地址'
            },
            {
                'name': 'remove_number',
                'zh_name': '去除数字',
                'description': '去除数字和字母数字标识符，如电话号码、信用卡号、十六进制散列等，同时跳过年份和简单数字的实例'
            }
        ]
    }
]
