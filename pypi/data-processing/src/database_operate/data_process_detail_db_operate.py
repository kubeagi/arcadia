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


def insert_transform_info(req_json, pool):
    """Insert a transform info"""
    now = date_time_utils.now_str()
    user = req_json["create_user"]
    program = "数据处理任务详情-新增"

    params = {
        "id": req_json.get("id"),
        "task_id": req_json.get("task_id"),
        "document_id": req_json.get("document_id"),
        "document_chunk_id": req_json.get("document_chunk_id"),
        "file_name": req_json.get("file_name"),
        "transform_type": req_json.get("transform_type"),
        "pre_content": req_json.get("pre_content"),
        "post_content": req_json.get("post_content"),
        "status": req_json.get("status"),
        "error_message": req_json.get("error_message"),
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_detail (
          id,
          task_id,
          document_id,
          document_chunk_id,
          file_name,
          transform_type,
          pre_content,
          post_content,
          status,
          error_message,
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
          %(document_id)s,
          %(document_chunk_id)s,
          %(file_name)s,
          %(transform_type)s,
          %(pre_content)s,
          %(post_content)s,
          %(status)s,
          %(error_message)s,
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


def insert_question_answer_info(req_json, pool):
    """Insert a question answer info"""
    now = date_time_utils.now_str()
    user = req_json["create_user"]
    program = "数据处理任务问题和答案-新增"

    params = {
        "id": req_json["id"],
        "task_id": req_json["task_id"],
        "document_id": req_json["document_id"],
        "document_chunk_id": req_json["document_chunk_id"],
        "file_name": req_json["file_name"],
        "question": req_json["question"],
        "answer": req_json["answer"],
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_question_answer (
          id,
          task_id,
          document_id,
          document_chunk_id,
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
          %(document_id)s,
          %(document_chunk_id)s,
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


def list_file_name_for_transform(req_json, pool):
    """List file name for transform in the task detail.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "transform_type": "remove_invisible_characters"
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
        from public.data_process_task_detail
      where 
      task_id = %(task_id)s and
      transform_type = %(transform_type)s
      group by file_name
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def top_n_list_transform_for_preview(req_json, pool):
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
        "task_id": req_json["task_id"],
        "file_name": req_json["file_name"],
        "transform_type": req_json["transform_type"],
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


def list_file_name_in_qa_by_task_id(req_json, pool):
    """List file name in question answer with task id.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["task_id"]}

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


def top_n_list_qa_for_preview(req_json, pool):
    """List question answer info with task id for preview.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "file_name": "MyFile.pdf"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["task_id"]}

    sql = """
        select
          id,
          task_id,
          file_name,
          question,
          answer,
          create_datetime,
          create_user,
          create_program,
          update_datetime,
          update_user,
          update_program
        from
          public.data_process_task_question_answer_clean
        where
          task_id = %(task_id)s and
          duplicated_flag = '1'
        order by random()
        limit 10
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def delete_transform_by_task_id(req_json, pool):
    """delete transform info by task id.

    req_json is a dictionary object. for example:
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["id"]}

    sql = """
        delete from public.data_process_task_detail
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
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
        delete from public.data_process_task_question_answer
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def list_file_name_for_clean(req_json, pool):
    """List file name for clean in the task detail.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["task_id"]}

    sql = """
      select 
        file_name 
      from public.data_process_task_detail
      where 
        task_id = %(task_id)s and
        transform_type in ('remove_invisible_characters', 'space_standardization', 'remove_garbled_text', 'traditional_to_simplified', 'remove_html_tag', 'remove_emojis')
      group by file_name
      """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def insert_question_answer_clean_info(req_json, pool):
    """Insert a question answer clean info"""
    now = date_time_utils.now_str()
    user = req_json["create_user"]
    program = "数据处理任务问题和答案-新增"

    params = {
        "id": req_json["id"],
        "task_id": req_json["task_id"],
        "document_id": req_json["document_id"],
        "document_chunk_id": req_json["document_chunk_id"],
        "file_name": req_json["file_name"],
        "question": req_json["question"],
        "answer": req_json["answer"],
        "question_score": req_json["question_score"],
        "answer_score": req_json["answer_score"],
        "duplicated_flag": req_json["duplicated_flag"],
        "compare_with_id": req_json["compare_with_id"],
        "create_datetime": now,
        "create_user": user,
        "create_program": program,
        "update_datetime": now,
        "update_user": user,
        "update_program": program,
    }

    sql = """
        insert into public.data_process_task_question_answer_clean (
          id,
          task_id,
          document_id,
          document_chunk_id,
          file_name,
          question,
          answer,
          question_score,
          answer_score,
          duplicated_flag,
          compare_with_id,
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
          %(document_id)s,
          %(document_chunk_id)s,
          %(file_name)s,
          %(question)s,
          %(answer)s,
          %(question_score)s,
          %(answer_score)s,
          %(duplicated_flag)s,
          %(compare_with_id)s,
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


def query_question_answer_list(document_id, pool):
    """List question answer with document id.

    req_json is a dictionary object. for example:
    {
        "document_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"document_id": document_id}

    sql = """
      select
        id,
        task_id,
        document_id,
        document_chunk_id,
        file_name,
        question,
        answer
      from public.data_process_task_question_answer
      where 
        document_id = %(document_id)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def list_file_name_for_privacy(req_json, pool):
    """List file name for privacy in the task detail.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["task_id"]}

    sql = """
      select 
        file_name 
      from public.data_process_task_detail
      where 
        task_id = %(task_id)s and
        transform_type in ('remove_email', 'space_standardization', 'remove_ip_address', 'remove_number')
      group by file_name
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def delete_qa_clean_by_task_id(req_json, pool):
    """delete qa clean info by task id.

    req_json is a dictionary object. for example:
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"task_id": req_json["id"]}

    sql = """
        delete from public.data_process_task_question_answer_clean
        where
          task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def list_for_transform_type(req_json, pool):
    """List transform for clean in the task detail."""
    params = {
        "task_id": req_json.get("task_id"),
        "document_id": req_json.get("document_id"),
        "transform_type": tuple(req_json.get("transform_type")),
    }

    sql = """
      select 
        id,
        task_id,
        file_name,
        transform_type,
        pre_content,
        post_content,
        status,
        error_message,
        update_datetime
      from public.data_process_task_detail
      where
        task_id = %(task_id)s and
        document_id = %(document_id)s and
        transform_type in %(transform_type)s
      """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res


def delete_transform_by_document_chunk(req_json, pool):
    """delete transform by task id and document id and chunk id.

    req_json is a dictionary object. for example:
    {
        "task_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "document_id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "document_chunk_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {
        "task_id": req_json.get("task_id"),
        "document_id": req_json.get("document_id"),
        "document_chunk_id": req_json.get("document_chunk_id"),
    }

    sql = """
        delete from public.data_process_task_detail
        where
          task_id = %(task_id)s and
          document_id = %(document_id)s and
          document_chunk_id = %(document_chunk_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res

def query_question_answer_clean_list(document_id, pool):
    """List question answer with document id.

    req_json is a dictionary object. for example:
    {
        "document_id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    pool: databasec connection pool;
    """
    params = {"document_id": document_id}

    sql = """
      select
        dptqac.id,
        dptqac.task_id,
        dptqac.document_id,
        dptqac.document_chunk_id,
        dptqac.file_name,
        dptqac.question,
        dptqac.answer,
        dptdc.content,
        dptdc.meta_info,
        dptdc.page_number
      from public.data_process_task_question_answer_clean dptqac
      left join public.data_process_task_document_chunk dptdc
      on
        dptdc.id = dptqac.document_chunk_id
      where 
        dptqac.document_id = %(document_id)s and
        dptqac.duplicated_flag = '1'
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res
