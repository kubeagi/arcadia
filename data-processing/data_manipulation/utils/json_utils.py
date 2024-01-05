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


from pathlib import Path

import ujson


def get_str_empty(
    json_item,
    json_key
):
    if json_item.get(json_key, '') is None:
        return ''

    return json_item.get(json_key, '')


def write_json_file(
    file_name,
    data,
    indent=None,
    ensure_ascii=None,
    escape_forward_slashes=None
):
    file_name = Path(file_name)
    with open(file_name, 'w', encoding='utf-8') as outfile:
        dump(
            data,
            outfile,
            indent,
            ensure_ascii,
            escape_forward_slashes
        )


def read_json_file(
    file_name
):
    file_name = Path(file_name)
    json_result = None
    with open(file_name, 'r', encoding='utf-8') as f:
        json_result = ujson.load(f)

    return json_result


def dumps(
    json_data,
    indent=None,
    ensure_ascii=None,
    sort_keys=None,
    escape_forward_slashes=None
):
    if indent is None:
        indent = 2
    if ensure_ascii is None:
        ensure_ascii = False
    if sort_keys is None:
        sort_keys = False
    if escape_forward_slashes is None:
        escape_forward_slashes = False

    return ujson.dumps(json_data,
                indent=indent,
                ensure_ascii=ensure_ascii,
                sort_keys=sort_keys,
                escape_forward_slashes=escape_forward_slashes)


def dump(
    json_data,
    file,
    indent=None,
    ensure_ascii=None,
    escape_forward_slashes=None
):
    if indent is None:
        indent=2
    if ensure_ascii is None:
        ensure_ascii=False
    if escape_forward_slashes is None:
        escape_forward_slashes=False

    ujson.dump(json_data,
               file,
               indent=indent,
               ensure_ascii=ensure_ascii,
               escape_forward_slashes=escape_forward_slashes)

