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


from database_clients import postgresql_pool_client
from utils import date_time_utils


def insert_transform_info(
    req_json,
    pool
):
    """Insert a transform info"""
    now = date_time_utils.now_str()
    user = req_json['create_user']
    program = '数据处理任务详情-新增'

    params = {
        'id': req_json['id'],
        'task_id': req_json['task_id'],
        'file_name': req_json['file_name'],
        'transform_type': req_json['transform_type'],
        'pre_content': req_json['pre_content'],
        'post_content': req_json['post_content'],
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

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def insert_question_answer_info(
    req_json,
    pool
):
    """Insert a question answer info"""
    now = date_time_utils.now_str()
    user = req_json['create_user']
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

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def list_file_name_for_transform(
    req_json,
    pool
):
    """List file name for transform in the task detail.
  
      req_json is a dictionary object. for example:
      {
          "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
          "transform_type": "remove_invisible_characters"
      }
      pool: databasec connection pool;
    """
    params = {
      'task_id': req_json['task_id'],
      'transform_type': req_json['transform_type'],
    }

    sql = """
      select 
        file_name 
        from public.data_process_task_detail
      where 
      task_id = %(task_id)s and
      transform_type = %(transform_type)s
      group by file_name
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def top_n_list_transform_for_preview(
    req_json,
    pool
):
    """List transform info with task id, file name and 
    transform type for preview.
    
    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "file_name": "MyFile.pdf",
        "transform_type": "remove_invisible_characters"
    }
    pool: databasec connection pool;
    """
    params = {
      'task_id': req_json['task_id'],
      'file_name': req_json['file_name'],
      'transform_type': req_json['transform_type']
    }

    sql = """
        select
          id,
          task_id,
          file_name,
          transform_type,
          pre_content,
          post_content,
          update_datetime
        from
          public.data_process_task_detail
        where
          task_id = %(task_id)s and 
          file_name = %(file_name)s and
          transform_type = %(transform_type)s
        order by update_datetime desc
        limit 10
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
  

def list_file_name_in_qa_by_task_id(
    req_json,
    pool
):
    """List file name in question answer with task id.
    
    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {
      'task_id': req_json['task_id']
    }

    sql = """
      select 
        file_name 
        from public.data_process_task_question_answer
      where 
      task_id = %(task_id)s
      group by file_name
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def top_n_list_qa_for_preview(
    req_json,
    pool
):
    """List question answer info with task id for preview.
    
    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "file_name": "MyFile.pdf"
    }
    pool: databasec connection pool;
    """
    params = {
      'task_id': req_json['task_id'],
      'file_name': req_json['file_name']
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
          task_id = %(task_id)s and 
          file_name = %(file_name)s
        order by update_datetime desc
        limit 10
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res