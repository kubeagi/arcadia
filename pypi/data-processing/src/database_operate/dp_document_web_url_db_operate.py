# Copyright 2024 KubeAGI.
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


def add(
    req_json,
    pool
):
    """Add a new record"""
    now = date_time_utils.now_str()
    user = req_json['creator']
    program = '数据处理URL-新增'

    params = {
        'id': req_json['id'],
        'document_id': req_json['document_id'],
        'level': req_json['level'],
        'web_url': req_json['web_url'],
        'title': req_json['title'],
        'description': req_json['description'],
        'content': req_json['content'],
        'content_clean': req_json['content_clean'],
        'language': req_json['language'],
        'status': req_json['status'],
        'error_message': req_json['error_message'],
        'task_id': req_json['task_id'],
        'create_datetime': now,
        'create_user': user,
        'create_program': program,
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        insert into public.data_process_task_document_web_url (
          id,
          document_id,
          level,
          web_url,
          title,
          description,
          content,
          content_clean,
          language,
          status,
          error_message,
          task_id,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        )
        values (
          %(id)s,
          %(document_id)s,
          %(level)s,
          %(web_url)s,
          %(title)s,
          %(description)s,
          %(content)s,
          %(content_clean)s,
          %(language)s,
          %(status)s,
          %(error_message)s,
          %(task_id)s,
          %(create_datetime)s,
          %(create_user)s,
          %(create_program)s,
          %(update_datetime)s,
          %(update_user)s,
          %(update_program)s
        )
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def update_by_id(
    req_json,
    pool
):
    """update a new record"""
    now = date_time_utils.now_str()
    user = req_json['creator']
    program = '数据处理URL-更新'

    params = {
        'id': req_json['id'],
        'document_id': req_json['document_id'],
        'level': req_json['level'],
        'web_url': req_json['web_url'],
        'title': req_json['title'],
        'description': req_json['description'],
        'content': req_json['content'],
        'content_clean': req_json['content_clean'],
        'language': req_json['language'],
        'status': req_json['status'],
        'error_message': req_json['error_message'],
        'task_id': req_json['task_id'],
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        update public.data_process_task_document_web_url set 
          web_url =  %(web_url)s,
          title =  %(title)s,
          description = %(description)s,
          content = %(content)s,
          content_clean = %(content_clean)s,
          language = %(language)s,
          status = %(status)s,
          error_message = %(error_message)s,
          update_datetime = %(update_datetime)s,
          update_user = %(update_user)s,
          update_program = %(update_program)s
        where id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def delete_by_id(
    req_json,
    pool
):
    """delete a record"""
    params = {
        'id': req_json['id']
    }

    sql = """
        delete from public.data_process_task_document_web_url 
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def info_by_id(
    req_json,
    pool
):
    """info with id"""
    params = {
        'id': req_json['id']
    }

    sql = """
        select
          id,
          document_id,
          level,
          web_url,
          title,
          description,
          content,
          content_clean,
          language,
          status,
          error_message,
          task_id,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        from
          public.data_process_task_document_web_url
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def list_by_count(
    req_json,
    pool
):
    """Get count for the list url with page"""
    params = {
        'keyword': '%' + req_json['web_url'] + '%'
    }

    sql = """
        select
          count(*)
        from
          public.data_process_task_document_web_url
        where
          web_url like %(keyword)s 
    """.strip()

    res = postgresql_pool_client.execute_count_query(pool, sql, params)
    return res


def list_by_page(
    req_json,
    pool
):
    """Get the list data for url by page"""
    params = {
        'keyword': '%' + req_json['title'] + '%',
        'pageIndex': int(req_json['pageIndex']),
        'pageSize': int(req_json['pageSize'])
    }

    sql = """
        select
          id,
          document_id,
          level,
          web_url,
          title,
          description,
          content,
          content_clean,
          language,
          status,
          error_message,
          task_id,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        from
          public.data_process_task_document_web_url
        where
          title like %(keyword)s 
        order by create_datetime desc
        limit %(pageSize)s offset %(pageIndex)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
