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
# MinIO
###

import urllib3

from minio import Minio

from common import config


async def create_client():
    return Minio(
        config.minio_api_url,
        access_key=config.minio_access_key,
        secret_key=config.minio_secret_key,
        secure=bool(config.minio_secure),
        http_client=urllib3.PoolManager(
            timeout=urllib3.Timeout.DEFAULT_TIMEOUT,
            cert_reqs='CERT_NONE',
            retries=urllib3.Retry(
                total=5,
                backoff_factor=0.2,
                status_forcelist=[500, 502, 503, 504],
            )
        )
    )
