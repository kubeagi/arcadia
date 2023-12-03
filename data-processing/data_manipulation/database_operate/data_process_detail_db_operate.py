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


import ulid

from database_clients import postgresql_pool_client
from utils import date_time_utils


async def insert_transform_info(req_json, opt={}):
    """Insert a transform info"""
    pool = opt['pool']
   
    now = date_time_utils.now_str()
    user = 'admin'
    program = '数据处理任务详情-新增'

    params = {
        'id': req_json['id'],
        'task_id': req_json['task_id'],
        'file_name': req_json['file_name'],
        'transform_type': req_json['transform_type'],
        'pre_content': req_json['pre_content'],
        'post_content': req_json['pre_content'],
        'create_datetime': now,
        'create_user': user,
        'create_program': program,
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        insert into public.data_process_task_detail (
          id,
          task_id,
          file_name,
          transform_type,
          pre_content,
          post_content,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_program,
          update_user
        )
        values (
          %(id)s,
          %(task_id)s,
          %(file_name)s,
          %(transform_type)s,
          %(pre_content)s,
          %(post_content)s,
          %(create_datetime)s,
          %(create_program)s,
          %(create_user)s,
          %(update_datetime)s,
          %(update_program)s,
          %(update_user)s 
        )
    """.strip()

    res =  await postgresql_pool_client.execute_update(pool, sql, params)
    return res


async def insert_question_answer_info(req_json, opt={}):
    """Insert a question answer info"""
    pool = opt['pool']
   
    now = date_time_utils.now_str()
    user = 'admin'
    program = '数据处理任务问题和答案-新增'

    params = {
        'id': req_json['id'],
        'task_id': req_json['task_id'],
        'file_name': req_json['file_name'],
        'question': req_json['question'],
        'answer': req_json['answer'],
        'create_datetime': now,
        'create_user': user,
        'create_program': program,
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        insert into public.data_process_task_question_answer (
          id,
          task_id,
          file_name,
          question,
          answer,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_program,
          update_user
        )
        values (
          %(id)s,
          %(task_id)s,
          %(file_name)s,
          %(question)s,
          %(answer)s,
          %(create_datetime)s,
          %(create_program)s,
          %(create_user)s,
          %(update_datetime)s,
          %(update_program)s,
          %(update_user)s 
        )
    """.strip()

    res =  await postgresql_pool_client.execute_update(pool, sql, params)
    return res


async def transform_list_by_task_id(req_json,opt={}):
    """Get list for the transform info with task id"""
    pool = opt['pool']

    params = {
        'task_id': req_json['task_id']
    }

    sql = """
        select
          id,
          task_id,
          file_name,
          transform_type,
          pre_content,
          post_content
        from
          public.data_process_task_detail
        where
          task_id = %(task_id)s
    """.strip()

    res = await postgresql_pool_client.execute_sql(pool,sql,params)
    return res


async def question_answer_info_by_task_id(req_json,opt={}):
    """ question answer info with task id"""
    pool = opt['pool']

    params = {
        'task_id': req_json['task_id']
    }

    sql = """
        select
          id,
          task_id,
          file_name,
          question,
          answer
        from
          public.data_process_task_question_answer
        where
          task_id = %(task_id)s
    """.strip()

    res = await postgresql_pool_client.execute_sql(pool,sql,params)
    return res

