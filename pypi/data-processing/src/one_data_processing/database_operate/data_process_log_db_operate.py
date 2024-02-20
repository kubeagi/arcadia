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
    program = "数据处理任务日志-新增"

    params = {
        "id": req_json.get("id"),
        "task_id": req_json.get("task_id"),
        "type": req_json.get("type"),
        "status": "processing",
        "error_msg": req_json.get("error_msg"),
        "start_datetime": now,
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_log (
          id,
          task_id,
          type,
          status,
          error_msg,
          start_datetime,
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
          %(type)s,
          %(status)s,
          %(error_msg)s,
          %(start_datetime)s,
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


def update_status_by_id(req_json, pool):
    """Update the status with id"""
    now = date_time_utils.now_str()
    user = req_json["creator"]
    program = "添加错误日志信息"

    params = {
        "id": req_json["id"],
        "status": req_json["status"],
        "error_msg": req_json["error_msg"],
        "end_datetime": now,
        "update_datetime": now,
        "update_program": program,
        "update_user": user,
    }

    sql = """
        update public.data_process_task_log set
          status = %(status)s,
          end_datetime = %(end_datetime)s,
          error_msg = %(error_msg)s,
          update_datetime = %(update_datetime)s,
          update_program = %(update_program)s,
          update_user = %(update_user)s
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
        delete from public.data_process_task_log
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res
