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


def dumps(
    json_data,
    indent=None,
    ensure_ascii=None,
    sort_keys=None,
    escape_forward_slashes=None,
):
    if indent is None:
        indent = 2
    if ensure_ascii is None:
        ensure_ascii = False
    if sort_keys is None:
        sort_keys = False
    if escape_forward_slashes is None:
        escape_forward_slashes = False

    return ujson.dumps(
        json_data,
        indent=indent,
        ensure_ascii=ensure_ascii,
        sort_keys=sort_keys,
        escape_forward_slashes=escape_forward_slashes,
    )

def loads(
    data,
):
    return ujson.loads(
        data,
    )
