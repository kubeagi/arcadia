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
minio_api_url = os.getenv('MINIO_API_URL', 'localhost:9000')
# 如果使用HTTP，将secure设置为False；如果使用HTTPS，将其设置为True
minio_secure = os.getenv('MINIO_SECURE', False)
minio_dataset_prefix = os.getenv('MINIO_DATASET_PREFIX', 'dataset')

# zhipuai api_key
zhipuai_api_key = os.getenv('ZHIPUAI_API_KEY', 'xxxxxx')

knowledge_chunk_size = os.getenv("KNOWLEDGE_CHUNK_SIZE", 500)
knowledge_chunk_overlap = os.getenv("KNOWLEDGE_CHUNK_OVERLAP", 50)

# pg数据库
pg_host = os.getenv("PG_HOST", "localhost")
pg_port = os.getenv("PG_PORT", 5432)
pg_user = os.getenv("PG_USER", "postgres")
pg_password = os.getenv("PG_PASSWORD", "xxxxxx")
pg_database = os.getenv("PG_DATABASE", "data_process")