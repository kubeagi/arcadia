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


def add(req_json, pool):
    """Add a new record"""
    now = date_time_utils.now_str()
    user = req_json["creator"]
    program = "数据处理文件拆分-新增"

    params = {
        "id": req_json.get("id"),
        "document_id": req_json.get("document_id"),
        "status": req_json.get("status"),
        "task_id": req_json.get("task_id"),
        "content": req_json.get("content"),
        "meta_info": req_json.get("meta_info"),
        "page_number": req_json.get("page_number"),
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_document_chunk (
          id,
          document_id,
          status,
          task_id,
          content,
          meta_info,
          page_number,
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
          %(status)s,
          %(task_id)s,
          %(content)s,
          %(meta_info)s,
          %(page_number)s,
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


def update_document_chunk_status_and_start_time(req_json, pool):
    """Update the status and start time with id"""
    now = req_json["start_time"]
    program = "开始处理chunk后的内容"

    params = {
        "id": req_json["id"],
        "status": req_json["status"],
        "start_time": now,
        "update_datetime": now,
        "update_user": req_json["update_user"],
        "update_program": program,
    }

    sql = """
        update public.data_process_task_document_chunk set
          status = %(status)s,
          start_time = %(start_time)s,
          update_datetime = %(update_datetime)s,
          update_user = %(update_user)s,
          update_program = %(update_program)s
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def update_document_chunk_status_and_end_time(req_json, pool):
    """Update the status and end time with id"""
    now = req_json["end_time"]
    program = "chunk后的内容处理完成"

    params = {
        "id": req_json["id"],
        "status": req_json["status"],
        "end_time": now,
        "update_datetime": now,
        "update_user": req_json["update_user"],
        "update_program": program,
    }

    sql = """
        update public.data_process_task_document_chunk set
          status = %(status)s,
          end_time = %(end_time)s,
          update_datetime = %(update_datetime)s,
          update_user = %(update_user)s,
          update_program = %(update_program)s
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def delete_by_task_id(req_json, pool):
    """delete info by task id.

    req_json is a dictionary object. for example:
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["id"]}

    sql = """
        delete from public.data_process_task_document_chunk
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def list_by_status(req_json, pool):
    """Retrieve a list of statuses marked as in progress and failed."""
    params = {"document_id": req_json.get("document_id")}

    sql = """
      select 
        id,
        document_id,
        status,
        task_id,
        content,
        meta_info,
        page_number,
        create_datetime,
        create_user,
        create_program,
        update_datetime,
        update_user,
        update_program
      from
        public.data_process_task_document_chunk
      where 
        document_id = %(document_id)s and
        status != 'success'
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
