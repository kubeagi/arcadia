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


def add(
    params,
    pool
):
    """Add a new record"""

    sql = """
        insert into public.data_process_task_question_answer_remove_duplicate_tmp (  
          id,
          task_id,
          document_id,
          document_chunk_id,
          file_name,
          question,
          question_vector,
          answer,
          answer_vector,
          create_datetime
        )
        values (
          %(id)s,
          %(task_id)s,
          %(document_id)s,
          %(document_chunk_id)s,
          %(file_name)s,
          %(question)s,
          %(question_vector)s,
          %(answer)s,
          %(answer_vector)s,
          %(create_datetime)s
        )
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res

def filter_by_distance(
    params,
    pool
):
    """Get the list data """

    sql = """
        select 
            id,
            task_id,
            document_id,
            document_chunk_id,
            file_name,
            question,
            answer,
            1 - (q1.question_vector <=> q2.question_vector) as question_distance,
            1 - (q1.answer_vector <=> q2.answer_vector) as answer_distance
        from 
            data_process_task_question_answer_remove_duplicate_tmp q1,
            (select
                question_vector,
                answer_vector
              from
                 data_process_task_question_answer_remove_duplicate_tmp
              where
                 id = %(id)s 
              limit 1) q2 
        where
            q1.task_id=%(task_id)s and
            q1.document_id = %(document_id)s and
            q1.id != %(id)s
    """.strip()

    res = postgresql_pool_client.execute_query(pool, sql, params)
    return res

def delete_by_task_document(
    params,
    pool
):
    """delete record by task and document"""

    sql = """
        delete from public.data_process_task_question_answer_remove_duplicate_tmp 
        where
          document_id = %(document_id)s
          and task_id = %(task_id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res


def delete_by_id(
    params,
    pool
):
    """delete record by id"""

    sql = """
        delete from public.data_process_task_question_answer_remove_duplicate_tmp 
        where
          id = %(id)s
    """.strip()

    res = postgresql_pool_client.execute_update(pool, sql, params)
    return res
