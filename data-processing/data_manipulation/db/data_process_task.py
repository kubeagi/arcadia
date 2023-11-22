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
# 数据处理任务
# @author: wangxinbiao
# @date: 2023-11-21 13:57:01
# modify history
# ==== 2023-11-21 13:57:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

from datetime import datetime

import ujson
import ulid
from sanic.response import json

from utils import pg_utils


async def list_by_page(request, opt={}):
    conn = opt['conn']

    req_json = request.json

    params = {
        'keyword': '%' + req_json['keyword'] + '%',
        'page': int(req_json['page']),
        'pageSize': int(req_json['pageSize'])
    }

    sql = """
        select
          id,
          name,
          status,
          pre_data_set_name,
          pre_data_set_version,
          post_data_set_name,
          post_data_set_version,
          start_datetime
        from
          public.data_process_task
        where
          name like %(keyword)s
        limit %(pageSize)s offset %(page)s
    """.strip()

    res = await pg_utils.execute_sql(conn,sql,params)
    return json(res)
  

async def list_by_count(request, opt={}):
    conn = opt['conn']

    req_json = request.json

    params = {
        'keyword': '%' + req_json['keyword'] + '%'
    }

    sql = """
        select
          count(*)
        from
          public.data_process_task
        where
          name like %(keyword)s
    """.strip()

    res = await pg_utils.execute_count_sql(conn,sql,params)
    return json(res)


async def add(request, opt={}):
    conn = opt['conn']

    req_json = request.json

    now = datetime.now()
    user = 'admin'
    program = '数据处理任务-新增'

    params = {
        'id': opt['id'],
        'name': req_json['name'],
        'file_type': req_json['file_type'],
        'status': 'processing',
        'pre_data_set_name': req_json['pre_data_set_name'],
        'pre_data_set_version': req_json['pre_data_set_version'],
        'file_names': ujson.dumps(req_json['file_names']),
        'post_data_set_name': req_json['post_data_set_name'],
        'post_data_set_version': req_json['post_data_set_version'],
        'data_process_config_info': ujson.dumps(req_json['data_process_config_info']),
        'start_datetime': now,
        'create_datetime': now,
        'create_user': user,
        'create_program': program,
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        insert into public.data_process_task (
          id,
          name,
          file_type,
          status,
          pre_data_set_name,
          pre_data_set_version,
          file_names,
          post_data_set_name,
          post_data_set_version,
          data_process_config_info,
          start_datetime,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        )
        values (
          %(id)s,
          %(name)s,
          %(file_type)s,
          %(status)s,
          %(pre_data_set_name)s,
          %(pre_data_set_version)s,
          %(file_names)s,
          %(post_data_set_name)s,
          %(post_data_set_version)s,
          %(data_process_config_info)s,
          %(start_datetime)s,
          %(create_datetime)s,
          %(create_program)s,
          %(create_user)s,
          %(update_datetime)s,
          %(update_program)s,
          %(update_user)s 
        )
    """.strip()

    return await pg_utils.execute_insert_sql(conn,sql,params)


async def delete_by_id(request, opt={}):
    conn = opt['conn']

    req_json = request.json

    params = {
        'id': req_json['id']
    }

    sql = """
        delete from public.data_process_task 
        where
          id = %(id)s
    """.strip()

    res =  await pg_utils.execute_delete_sql(conn,sql,params)
    return json(res)


async def update_status_by_id(opt={}):
    conn = opt['conn']

    now = datetime.now()
    user = 'admin'
    program = '修改任务状态'

    params = {
        'id': opt['id'],
        'status': opt['status'],
        'update_datetime': now,
        'update_program': program,
        'update_user': user
    }

    sql = """
        UPDATE public.dataset set
          status = %(status)s
          update_datetime = %(update_datetime)s,
          update_program = %(update_program)s,
          update_user = %(update_user)s
        WHERE
          id = %(id)s
    """.strip()

    res =  await pg_utils.execute_update_sql(conn,sql,params)
    return json(res)
