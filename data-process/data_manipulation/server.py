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
# 数据处理 后端
# @author: wangxinbiao
# @date: 2023-10-31 17:34:01
# modify history
# ==== 2023-10-31 17:34:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

from sanic import Sanic
from sanic.response import json, text
from sanic_cors import CORS, cross_origin
from sanic.exceptions import NotFound

import asyncio
import aiohttp

import sys

import logging


from service import (
    minio_store_process_service
)

from transform.text import (
    support_type
)

from utils import (
    log_utils
)

###
# 初始化日志配置
###
log_utils.init_config({
    'source_type': 'manipulate_server'
})


logger = logging.getLogger('manipulate_server')

app = Sanic(name='data_manipulate')
CORS(app)

app.config['REQUEST_MAX_SIZE'] = 1024 * 1024 * 1024 # 1G
app.config['REQUEST_TIMEOUT'] = 60 * 60 * 60
app.config['RESPONSE_TIMEOUT'] = 60 * 60 * 60
app.config['KEEP_ALIVE_TIMEOUT'] = 60 * 60 * 60

###
# 文本数据处理
# @author: wangxinbiao
# @date: 2023-11-01 10:44:01
# modify history
# ==== 2023-11-01 10:44:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
@app.route('text-manipulate', methods=['POST'])
async def text_manipulate(request):
    """
        对文本类数据进行处理

        Args:
            type: 对文本数据需要进行那些处理;
            file_path: 文本路径

        Returns:
            
    """

    await asyncio.create_task(
        minio_store_process_service.text_manipulate(request)
    )

    return json({
        'status': 200,
        'message': '',
        'data': ''
    })

###
# 数据处理支持类型
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
@app.route('text-process-type', methods=['POST'])
async def text_process_type(request):
    """
        获取数据处理支持的类型

        Args:
            

        Returns:
            json: 支持的类型
    """

    return json({
        'status': 200,
        'message': '',
        'data': support_type.support_types
    })
    

if __name__ == '__main__':
    app.run(host='0.0.0.0',
            port=28888,
            access_log=True,
            debug=True,
            workers=2)