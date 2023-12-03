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


@data_process.route('list-by-page', methods=['POST'])
async def list_by_page(request):
    res = await data_process_service.list_by_page(request.json, {
        'pool': request.app.config['conn_pool']
    })
    return json(res)


@data_process.route('list-by-count', methods=['POST'])
async def list_by_count(request):
    res = await data_process_service.list_by_count(request.json, {
        'pool': request.app.config['conn_pool']
    })
    return json(res)


@data_process.route('add', methods=['POST'])
async def add(request):
    res = await data_process_service.add(request.json, {
        'pool': request.app.config['conn_pool'],
        'sanic_app': app
    })
    return json(res)


@data_process.route('delete-by-id', methods=['POST'])
async def delete_by_id(request):
    res = await data_process_service.delete_by_id(request.json, {
        'pool': request.app.config['conn_pool']
    })
    return json(res)


@data_process.route('info-by-id', methods=['POST'])
async def info_by_id(request):
    res = await data_process_service.info_by_id(request.json, {
        'pool': request.app.config['conn_pool']
    })
    return json(res) 


@data_process.route('text-process-type', methods=['POST'])
async def text_process_type(request):
    """Get the support type for transforming the text content."""
    return json({
        'status': 200,
        'message': '',
        'data': support_type.support_types
    })    
