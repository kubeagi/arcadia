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


def insert(req_json, pool):
    """Add a new record"""
    now = date_time_utils.now_str()
    user = req_json["creator"]
    program = "数据处理任务阶段日志-新增"

    params = {
        "id": ulid.ulid(),
        "task_id": req_json.get("task_id"),
        "log_id": req_json.get("log_id"),
        "log_datetime": now,
        "file_name": req_json.get("file_name"),
        "stage_name": req_json.get("stage_name"),
        "stage_status": req_json.get("stage_status"),
        "stage_detail": req_json.get("stage_detail"),
        "error_msg": req_json.get("error_msg"),
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_stage_log (
          id,
          task_id,
          log_id,
          log_datetime,
          file_name,
          stage_name,
          stage_status,
          stage_detail,
          error_msg,
          create_datetime,
          create_program,
          create_user,
          update_datetime,
          update_program,
          update_user
        )
        values (
          %(id)s,
          %(task_id)s,
          %(log_id)s,
          %(log_datetime)s,
          %(file_name)s,
          %(stage_name)s,
          %(stage_status)s,
          %(stage_detail)s,
          %(error_msg)s,
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


def list_by_task_id(req_json, pool):
    """Get the list data for data processing log by task id"""
    params = {"task_id": req_json.get("id")}

    sql = """
        select
          id,
          task_id,
          log_id,
          log_datetime,
          stage_name,
          stage_status,
          stage_detail,
          error_msg
        from
          public.data_process_task_stage_log
        where
          task_id = %(task_id)s
        order by log_datetime asc
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
        delete from public.data_process_task_stage_log
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def info_by_stage_and_file_name(req_json, pool):
    params = {
        "task_id": req_json.get("id"),
        "stage_name": req_json.get("type"),
        "file_name": req_json.get("file_name"),
    }

    sql = """
        select
          id,
          task_id,
          log_id,
          log_datetime,
          stage_name,
          stage_status,
          stage_detail,
          error_msg
        from
          public.data_process_task_stage_log
        where
          task_id = %(task_id)s and
          stage_name = %(stage_name)s and
          file_name = %(file_name)s
        order by log_datetime desc
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
