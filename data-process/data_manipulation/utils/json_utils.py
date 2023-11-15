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


import ujson
from pathlib import Path


def pretty_print(opt={}):
    data = opt.get('data', {})

    print(ujson.dumps(data,
                     ensure_ascii=False,
                     escape_forward_slashes=False,
                     indent=4))


def get_str_empty(opt={}):
    json_item = opt['json_item']
    json_key = opt['json_key']

    if json_item.get(json_key, '') is None:
        return ''

    return json_item.get(json_key, '')


def write_json_file(opt={}):
    file_name = Path(opt['file_name'])
    with open(file_name, 'w', encoding = 'utf-8') as outfile:
         dump(opt['data'], outfile, opt)


def read_json_file(opt={}):
    file_name = Path(opt['file_name'])
    json_result = None
    with open(file_name, 'r', encoding = 'utf-8') as f:
        json_result = ujson.load(f)

    return json_result




def dumps(json_data, opt={}):
    indent = opt.get('indent', 2)
    ensure_ascii = opt.get('ensure_ascii', False)
    escape_forward_slashes = opt.get('escape_forward_slashes', False)

    ujson.dumps(json_data,
                indent=indent,
                ensure_ascii=ensure_ascii,
                escape_forward_slashes=escape_forward_slashes)


def dump(json_data, file, opt={}):
    indent = opt.get('indent', 2)
    ensure_ascii = opt.get('ensure_ascii', False)
    escape_forward_slashes = opt.get('escape_forward_slashes', False)

    ujson.dump(json_data,
               file,
               indent=indent,
               ensure_ascii=ensure_ascii,
               escape_forward_slashes=escape_forward_slashes)