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

import csv
import logging
import os

from common import log_tag_const
from utils import file_utils

logger = logging.getLogger(__name__)


def save_csv(file_name, phase_value, data):
    """Save the csv file.

    file_name: file name;
    phase_value: phase value
    """
    csv_file_path = file_utils.get_temp_file_path()

    # 如果文件夹不存在，则创建
    directory_path = csv_file_path + phase_value
    if not os.path.exists(directory_path):
        os.makedirs(directory_path)

    file_path = directory_path + "/" + file_name

    logger.debug(
        "".join(
            [
                f"{log_tag_const.CSV_HANDLE} Save a csv file.\n",
                f"file path: {file_path}",
            ]
        )
    )

    with open(file_path, "w", newline="") as file:
        writer = csv.writer(file)
        writer.writerows(data)

    return file_path
