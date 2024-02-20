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


from sanic import Blueprint
from sanic.response import json

from service import data_process_service
from transform.text import support_type

# Create a separate router (Blueprint)
data_process = Blueprint("data_process", url_prefix="/")


@data_process.route("list-by-page", methods=["POST"])
async def list_by_page(request):
    res = data_process_service.list_by_page(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("list-by-count", methods=["POST"])
async def list_by_count(request):
    res = data_process_service.list_by_count(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("add", methods=["POST"])
async def add(request):
    """Add a new data process task.

    example for request.json
    {
        "name": "小T_test_0201",
        "file_type": "text",
        "pre_data_set_name": "dataset1",
        "pre_data_set_version": "v2",
        "post_data_set_name": "dataset1",
        "post_data_set_version": "v2",
        "version_data_set_name": "dataset1-v2",
        "file_names": [
            {
                "name": "数据处理文件_小T.pdf"
            }
        ],
        "data_process_config_info": [],
        "creator": "",
        "namespace": "abc"
    }
    """
    res = data_process_service.add(request.json, pool=request.app.config["conn_pool"])
    return json(res)


@data_process.route("delete-by-id", methods=["POST"])
async def delete_by_id(request):
    res = data_process_service.delete_by_id(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("info-by-id", methods=["POST"])
async def info_by_id(request):
    """Get the detail info by id.

    example for request.json
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    """
    res = data_process_service.info_by_id(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("text-process-type", methods=["POST"])
async def text_process_type(_request):
    """Get the support type for transforming the text content."""
    return json(
        {"status": 200, "message": "", "data": support_type.get_default_support_types()}
    )


@data_process.route("check-task-name", methods=["POST"])
async def check_task_name(request):
    """check task name by name and namespace.

    example for request.json
    {
        "name": "test",
        "namespace": "arcadia"
    }
    """
    res = data_process_service.check_task_name(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("get-log-info", methods=["POST"])
async def get_log_info(request):
    """check task name by name and namespace.

    example for request.json
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    """
    res = data_process_service.get_log_info(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("get-log-by-file-name", methods=["POST"])
async def get_log_by_file_name(request):
    """check task name by name and namespace.

    example for request.json
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS",
        "file_name": "xx.pdf",
        "type": "qa_split"
    }
    """
    res = data_process_service.get_log_by_file_name(
        request.json, pool=request.app.config["conn_pool"]
    )
    return json(res)


@data_process.route("retry", methods=["POST"])
async def retry(request):
    """check task name by name and namespace.

    example for request.json
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    """
    res = data_process_service.retry(request.json, pool=request.app.config["conn_pool"])
    return json(res)
