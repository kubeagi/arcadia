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


def insert(req_json, pool):
    """Insert info"""

    params = {
        "id": req_json["id"],
        "task_id": req_json["task_id"],
        "file_name": req_json["file_name"],
        "transform_type": req_json["transform_type"],
        "pre_content": req_json["pre_content"],
        "post_content": req_json["post_content"],
        "create_datetime": req_json["create_datetime"],
        "create_user": req_json["create_user"],
        "create_program": req_json["create_program"],
        "update_datetime": req_json["update_datetime"],
        "update_user": req_json["update_user"],
        "update_program": req_json["update_program"],
    }

    sql = """
        insert into public.data_process_task_detail_preview (
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


def list_file_name_by_task_id(req_json, pool):
    """List file name with task id and transform_type.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "transform_type": "qa_split"
    }
    pool: databasec connection pool;
    """
    params = {
        "task_id": req_json["task_id"],
        "transform_type": req_json["transform_type"],
    }

    sql = """
      select 
        file_name 
      from public.data_process_task_detail_preview
      where 
        task_id = %(task_id)s and
        transform_type = %(transform_type)s
      group by file_name
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def list_for_preview(req_json, pool):
    """List file name with task id and transform_type.

    req_json is a dictionary object. for example:
    {
      "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
      "transform_type": "qa_split"
    }
    pool: databasec connection pool;
    """
    params = {
        "task_id": req_json["task_id"],
        "transform_type": req_json["transform_type"],
    }

    sql = """
      select 
        id,
        task_id,
        file_name,
        transform_type,
        pre_content,
        post_content 
      from public.data_process_task_detail_preview
      where
        task_id = %(task_id)s and
        transform_type = %(transform_type)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def delete_qa_by_task_id(req_json, pool):
    """delete qa info by task id.

    req_json is a dictionary object. for example:
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["id"]}

    sql = """
        delete from public.data_process_task_detail_preview
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res
