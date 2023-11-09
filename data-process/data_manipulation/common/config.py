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

import os

minio_access_key = os.getenv('MINIO_ACCESSKEY', 'minioadmin')
minio_secret_key = os.getenv('MINIO_SECRETKEY', 'minioadmin')
minio_api_url = os.getenv('MINIO_API_URL', '192.168.90.31:9000')
# 如果使用HTTP，将secure设置为False；如果使用HTTPS，将其设置为True
minio_secure = os.getenv('MINIO_SECURE', False)
