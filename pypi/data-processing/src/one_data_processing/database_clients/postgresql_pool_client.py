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

import logging
import traceback

import psycopg2.extras
from dbutils.pooled_db import PooledDB

from common import log_tag_const

logger = logging.getLogger(__name__)


def get_pool(connection_creator):
    """Get a database connection pool."""
    logger.debug(f"{log_tag_const.DATABASE_POSTGRESQL} Get a database connection pool.")

    return PooledDB(
        creator=connection_creator,
        maxcached=8,
        maxshared=8,
        maxconnections=8,
        blocking=True,
    )


def release_pool(pool):
    """Release the database connection pool."""
    if pool is not None:
        pool.close()
        logger.debug(
            f"{log_tag_const.DATABASE_POSTGRESQL} Release the database connection pool."
        )
    else:
        logger.debug(
            f"{log_tag_const.DATABASE_POSTGRESQL} The database connection pool is None."
        )


def get_connection_from_pool(pool):
    """Get a connection from the pool"""
    logger.debug(f"{log_tag_const.DATABASE_POSTGRESQL} Get a connection from the pool.")
    return pool.connection()


def execute_query(pool, sql, params):
    """Execute a query with the parameters."""
    error = ""
    data = []
    try:
        with pool.connection() as conn:
            with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cursor:
                cursor.execute(sql, params)
                result = cursor.fetchall()

                for row in result:
                    data_item = {}
                    for key in row.keys():
                        data_item[key] = row[key]
                    data.append(data_item)

    except Exception as ex:
        error = str(ex)
        data = None
        logger.error(
            "".join(
                [
                    f"{log_tag_const.DATABASE_POSTGRESQL} Executing the sql failed\n {sql} \n",
                    f"The error is: \n{error}\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

    if len(error) > 0:
        return {"status": 400, "message": error, "data": traceback.format_exc()}

    return {"status": 200, "message": "", "data": data}


def execute_count_query(pool, sql, params):
    """Execute a count query with the parameters."""
    error = ""
    data = None
    try:
        with pool.connection() as conn:
            with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cursor:
                cursor.execute(sql, params)
                data = cursor.fetchone()[0]
    except Exception as ex:
        error = str(ex)
        data = None
        logger.error(
            "".join(
                [
                    f"{log_tag_const.DATABASE_POSTGRESQL} Executing the count sql failed\n {sql} \n",
                    f"\nThe error is: \n{error}\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

    if len(error) > 0:
        return {"status": 400, "message": error, "data": traceback.format_exc()}

    return {"status": 200, "message": "", "data": data}


def execute_update(pool, sql, params):
    """Execute a update with the parameters."""
    error = ""
    data = None
    try:
        with pool.connection() as conn:
            with conn.cursor() as cursor:
                cursor.execute(sql, params)
                conn.commit()
    except Exception as ex:
        error = str(ex)
        data = None
        conn.rollback()
        logger.error(
            "".join(
                [
                    f"{log_tag_const.DATABASE_POSTGRESQL} Executing the update sql failed\n {sql} \n",
                    f"\nThe error is: \n{error}\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

    if len(error) > 0:
        return {"status": 400, "message": error, "data": traceback.format_exc()}

    return {"status": 200, "message": "处理成功", "data": data}
