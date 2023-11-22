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


###
# pg数据库工具类
###

import psycopg2.extras

async def execute_sql(conn,sql,record_to_select):
    '''
    执行sql语句
    :param conn:
    :param sql:
    :return:
    '''
    error = ''
    data = []
    try:
        cursor = conn.cursor(cursor_factory=psycopg2.extras.DictCursor)
        cursor.execute(sql,record_to_select)
        result = cursor.fetchall()
        for row in result:
            dataItem = {}
            for item in row.keys():
                dataItem[item] = row[item]
            data.append(dataItem)
        
        conn.commit()
        cursor.close()
    except Exception as ex:
        print('查询', ex)
        conn.rollback()
        error = str(ex)
        data = None

    if len(error) > 0:
        return {
            'status': 400,
            'message': error,
            'data': None
        }

    return {
        'status': 200,
        'message': '',
        "data": data
    }

async def execute_count_sql(conn,sql,record_to_select_count):
    '''
    执行count sql语句
    :param conn:
    :param sql:
    :return:
    '''
    error = ''
    number_count = 0
    try:
        cursor = conn.cursor()
        cursor.execute(sql,record_to_select_count)
        result = cursor.fetchall()
        for row in result:
            number_count = row
        
        conn.commit()
        cursor.close()
    except Exception as ex:
        print('mysql 查询', ex)
        conn.rollback()
        error = str(ex)
        data = None

    if len(error) > 0:
        return {
            'status': 400,
            'message': error,
            'data': None
        }
    return {
        'status': 200,
        'message': '',
        "data": number_count[0]
    }

async def execute_insert_sql(conn,sql,record_to_insert):
    '''
    执行insert sql语句
    :param conn:
    :param sql:
    :return:
    '''
    error = ''
    try:
        cursor = conn.cursor()
        cursor.execute(sql,record_to_insert)
        
        conn.commit()
        cursor.close()
    except Exception as ex:
        print('insert 失败', ex)
        conn.rollback()
        error = str(ex)
        data = None

    if len(error) > 0:
        return {
            'status': 400,
            'message': error,
            'data': None
        }

    return {
        'status': 200,
        'message': '新增成功',
        "data": None
    }

async def execute_update_sql(conn,sql,record_to_update):
    '''
    执行 update sql语句
    :param conn:
    :param sql:
    :return:
    '''
    error = ''
    number_count = 0
    try:
        cursor = conn.cursor()
        cursor.execute(sql,record_to_update)
        
        conn.commit()
        cursor.close()
    except Exception as ex:
        print('update 失败', ex)
        conn.rollback()
        error = str(ex)
        data = None

    if len(error) > 0:
        return {
            'status': 400,
            'message': error,
            'data': None
        }

    return {
        'status': 200,
        'message': '编辑成功',
        "data": None
    }

async def execute_delete_sql(conn,sql,record_to_delete):
    '''
    执行 delete sql语句
    :param conn:
    :param sql:
    :return:
    '''
    error = ''
    number_count = 0
    try:
        cursor = conn.cursor()
        cursor.execute(sql,record_to_delete)
        
        conn.commit()
        cursor.close()
    except Exception as ex:
        print('delete 失败', ex)
        conn.rollback()
        error = str(ex)
        data = None

    if len(error) > 0:
        return {
            'status': 400,
            'message': error,
            'data': None
        }

    return {
        'status': 200,
        'message': '删除成功',
        "data": None
    }
