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
        'task_id': req_json['task_id'],
        'url': req_json['url'],
        'image_path': req_json['image_path'],
        'ocr_content': req_json['ocr_content'],
        'image_info': req_json['image_info'],
        'meta_info': req_json['meta_info'],
        'create_datetime': now,
        'create_user': user,
        'create_program': program,
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        insert into public.data_process_task_document_image (
          id,
          document_id,
          task_id,
          url,
          image_path,
          ocr_content,
          image_info,
          meta_info,
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
          %(task_id)s,
          %(url)s,
          %(image_path)s,
          %(ocr_content)s,
          %(image_info)s,
          %(meta_info)s,
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
        'task_id': req_json['task_id'],
        'url': req_json['url'],
        'image_path': req_json['image_path'],
        'ocr_content': req_json['ocr_content'],
        'image_info': req_json['image_info'],
        'meta_info': req_json['meta_info'],
        'update_datetime': now,
        'update_user': user,
        'update_program': program
    }

    sql = """
        update public.data_process_task_document_image set 
          url =  %(url)s,
          image_path =  %(image_path)s,
          ocr_content = %(ocr_content)s,
          image_info = %(image_info)s,
          meta_info = %(meta_info)s
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
        delete from public.data_process_task_document_image 
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
          task_id,
          url,
          image_path,
          ocr_content,
          image_info,
          meta_info,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        from
          public.data_process_task_document_image
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
        'keyword': '%' + req_json['url'] + '%'
    }

    sql = """
        select
          count(*)
        from
          public.data_process_task_document_image
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
        'keyword': '%' + req_json['url'] + '%',
        'pageIndex': int(req_json['pageIndex']),
        'pageSize': int(req_json['pageSize'])
    }

    sql = """
        select
          id,
          document_id,
          task_id,
          url,
          image_path,
          ocr_content,
          image_info,
          meta_info,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        from
          public.data_process_task_document_image
        where
          url like %(keyword)s 
        order by create_datetime desc
        limit %(pageSize)s offset %(pageIndex)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
