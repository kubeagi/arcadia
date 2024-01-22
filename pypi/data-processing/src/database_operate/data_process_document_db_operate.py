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
    program = "数据处理文件进度-新增"

    params = {
        "id": req_json.get("id"),
        "file_name": req_json.get("file_name"),
        "status": req_json.get("status"),
        "progress": req_json.get("progress"),
        "task_id": req_json.get("task_id"),
        "from_source_type": req_json.get("from_source_type"),
        "from_source_path": req_json.get("from_source_path"),
        "document_type": req_json.get("document_type"),
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_document (
          id,
          file_name,
          status,
          progress,
          task_id,
          from_source_type,
          from_source_path,
          document_type,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        )
        values (
          %(id)s,
          %(file_name)s,
          %(status)s,
          %(progress)s,
          %(task_id)s,
          %(from_source_type)s,
          %(from_source_path)s,
          %(document_type)s,
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


def update_document_status_and_start_time(req_json, pool):
    """Update the status and start time with id"""
    now = req_json["start_time"]
    program = "文件开始处理-修改"

    params = {
        "id": req_json["id"],
        "status": req_json["status"],
        "start_time": now,
        "chunk_size": req_json["chunk_size"],
        "update_datetime": now,
        "update_program": program,
    }

    sql = """
        update public.data_process_task_document set
          status = %(status)s,
          start_time = %(start_time)s,
          chunk_size = %(chunk_size)s,
          update_datetime = %(update_datetime)s,
          update_program = %(update_program)s
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def update_document_status_and_end_time(req_json, pool):
    """Update the status and end time with id"""
    now = req_json["end_time"]
    program = "文件处理完成-修改"

    params = {
        "id": req_json["id"],
        "status": req_json["status"],
        "end_time": now,
        "update_datetime": now,
        "update_program": program,
    }

    sql = """
        update public.data_process_task_document set
          status = %(status)s,
          end_time = %(end_time)s,
          update_datetime = %(update_datetime)s,
          update_program = %(update_program)s
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def update_document_progress(req_json, pool):
    """Update the progress with id"""
    now = date_time_utils.now_str()
    program = "文件处理进度-修改"

    params = {
        "id": req_json["id"],
        "progress": req_json["progress"],
        "update_datetime": now,
        "update_user": req_json["update_user"],
        "update_program": program,
    }

    sql = """
        update public.data_process_task_document set
          progress = %(progress)s,
          update_datetime = %(update_datetime)s,
          update_program = %(update_program)s
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def list_file_by_task_id(req_json, pool):
    """info with id"""
    params = {"task_id": req_json["task_id"]}

    sql = """
        select
          id,
          file_name,
          status,
          start_time,
          end_time,
          progress
        from
          public.data_process_task_document
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
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
        delete from public.data_process_task_document
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def list_by_task_id_and_status(req_json, pool):
    """info with task id and status"""
    params = {"task_id": req_json.get("id")}

    sql = """
        select
          id,
          file_name,
          status,
          start_time,
          end_time,
          progress,
          chunk_size,
          task_id,
          from_source_type,
          from_source_path,
          document_type
        from
          public.data_process_task_document
        where
          task_id = %(task_id)s and
          status != 'success'
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
