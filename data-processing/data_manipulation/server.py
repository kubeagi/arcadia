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

import asyncio
import logging

import psycopg2
from sanic import Sanic
from sanic.response import json
from sanic_cors import CORS

from common import config
# from kube import client
from service import data_process_service, minio_store_process_service
from transform.text import support_type
from utils import log_utils

###
# Initialize kubernetes client
###
# kube = client.KubeEnv()
# have a try!
# print(kube.list_versioneddatasets("arcadia"))


###
# 初始化日志配置
###
log_utils.init_config({
    'source_type': 'manipulate_server',
    'log_dir': "log"
})


logger = logging.getLogger('manipulate_server')

app = Sanic(name='data_manipulate')
CORS(app)

app.config['REQUEST_MAX_SIZE'] = 1024 * 1024 * 1024  # 1G
app.config['REQUEST_TIMEOUT'] = 60 * 60 * 60
app.config['RESPONSE_TIMEOUT'] = 60 * 60 * 60
app.config['KEEP_ALIVE_TIMEOUT'] = 60 * 60 * 60


@app.listener('before_server_start')
async def init_web_server(app, loop):
    app.config['conn'] = get_connection()


###
# 分页查询列表
# @author: wangxinbiao
# @date: 2023-11-21 11:31:01
# modify history
# ==== 2023-11-21 11:31:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


@app.route('list-by-page', methods=['POST'])
async def list_by_page(request):
    return await data_process_service.list_by_page(request, {
        'conn': app.config['conn']
    })

###
# 列表总记录数
# @author: wangxinbiao
# @date: 2023-11-21 15:45:01
# modify history
# ==== 2023-11-21 15:45:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


@app.route('list-by-count', methods=['POST'])
async def list_by_count(request):
    return await data_process_service.list_by_count(request, {
        'conn': app.config['conn']
    })

###
# 新增
# @author: wangxinbiao
# @date: 2023-11-21 15:45:01
# modify history
# ==== 2023-11-21 15:45:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


@app.route('add', methods=['POST'])
async def add(request):
    return await data_process_service.add(request, {
        'conn': app.config['conn']
    })

###
# 删除
# @author: wangxinbiao
# @date: 2023-11-21 15:45:01
# modify history
# ==== 2023-11-21 15:45:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


@app.route('delete-by-id', methods=['POST'])
async def delete_by_id(request):
    return await data_process_service.delete_by_id(request, {
        'conn': app.config['conn']
    })

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
            bucket_name: minio桶名称;
            folder_prefix: minio中文件目录

        Returns:

    """

    asyncio.create_task(
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


###
# 数据库链接
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


def get_connection():
    '''
    获取postgresql连接
    :param host:
    :param port:
    :param user:
    :param password:
    :param database:
    :return:
    '''
    conn = psycopg2.connect(database=config.pg_database, user=config.pg_user,
                            password=config.pg_password, host=config.pg_host, port=config.pg_port)

    # while True:
    #     cur = conn.cursor()
    #     cur.execute("SELECT 1")
    #     cur.close()
    #     time.sleep(3600)  # 每隔5分钟发送一次查询

    return conn


if __name__ == '__main__':
    app.run(host='0.0.0.0',
            port=28888,
            access_log=True,
            debug=True,
            workers=2)
