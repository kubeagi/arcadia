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
import os
import traceback

import urllib3
from minio import Minio
from minio.commonconfig import Tags
from minio.error import S3Error

from common import log_tag_const
from common.config import config
from utils import file_utils

logger = logging.getLogger(__name__)


def get_minio_client():
    """Get a new minio client."""
    return Minio(
        config.minio_api_url,
        access_key=config.minio_access_key,
        secret_key=config.minio_secret_key,
        secure=bool(config.minio_secure),
        http_client=urllib3.PoolManager(
            timeout=urllib3.Timeout.DEFAULT_TIMEOUT,
            cert_reqs="CERT_NONE",
            retries=urllib3.Retry(
                total=5,
                backoff_factor=0.2,
                status_forcelist=[500, 502, 503, 504],
            ),
        ),
    )


def download(
    minio_client,
    folder_prefix,
    bucket_name,
    file_name,
):
    """Download a file.

    minio_client: minio client;
    folder_prefix: folder prefix;
    bucket_name: bucket name;
    file_name: file name;
    """
    file_path = file_utils.get_temp_file_path()

    # 如果文件夹不存在，则创建
    directory_path = file_path + "original"
    if not os.path.exists(directory_path):
        os.makedirs(directory_path)

    file_path = directory_path + "/" + file_name

    minio_client.fget_object(bucket_name, folder_prefix + "/" + file_name, file_path)


def upload_files_to_minio_with_tags(
    minio_client,
    local_folder,
    minio_bucket,
    minio_prefix,
    support_type,
    data_volumes_file,
):
    """Upload the files to minio with tags

    local_folder: local folder;
    minio_bucket: bucket name;
    minio_prefix: folder prefix;
    support_type: support type
    data_volumes_file: data volumes file
    """

    logger.debug(f"{log_tag_const.MINIO} 上传文件到minio中 {data_volumes_file}")

    # 设置tag信息
    tags = Tags(for_object=True)
    tags["phase"] = "final"

    for root, _, files in os.walk(local_folder):
        for file in files:
            local_file_path = os.path.join(root, file)
            minio_object_name = os.path.join(
                minio_prefix, os.path.relpath(local_file_path, local_folder)
            )

            # 针对QA拆分类型的处理需要加上object_type和object_count标签
            if any(d.get("type") == "qa_split" for d in support_type):
                tags["object_type"] = "QA"

                filtered = [
                    item["object_count"]
                    for item in data_volumes_file
                    if item["object_name"] == file
                ]
                if filtered:
                    tags["object_count"] = str(filtered[0])

            try:
                minio_client.fput_object(
                    minio_bucket, minio_object_name, local_file_path, tags=tags
                )

                # 删除本地文件
                file_utils.delete_file(local_file_path)
            except S3Error as ex:
                logger.error(
                    "".join(
                        [
                            f"{log_tag_const.MINIO} Error uploading {minio_object_name} ",
                            f"The error is: \n{str(ex)}\n",
                            f"to {minio_bucket}. \n{traceback.format_exc()}",
                        ]
                    )
                )
