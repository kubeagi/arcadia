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

import pandas as pd
import ulid
from common import log_tag_const
from transform.text import clean_transform
from utils import csv_utils, file_utils

logger = logging.getLogger(__name__)


def text_manipulate(file_name, support_type):
    """Manipuate the text content.

    file_name: file name;
    support_type: support type;

    process logic
    处理某条数据时，如果某个方式（比如：去除不可见字符）处理失败了，则直接结束，不在处理，
    整个文件都视作处理失败。
    """
    try:
        logger.debug(
            f"{log_tag_const.CSV_HANDLE} Start to manipulate text in csv file."
        )

        csv_file_path = file_utils.get_temp_file_path()
        file_path = csv_file_path + "original/" + file_name

        # 获取CSV文件的内容
        data = pd.read_csv(file_path)
        text_data = data["prompt"]

        # 数据清洗
        clean_result = _data_clean(
            support_type=support_type, data=text_data, file_name=file_name
        )

        if clean_result["status"] != 200:
            return clean_result

        text_data = clean_result["data"]

        # 将清洗后的文件保存为final
        new_file_name = file_utils.get_file_name(
            {"file_name": file_name, "handle_name": "final"}
        )

        csv_utils.save_csv(
            {"file_name": new_file_name, "phase_value": "final", "data": text_data}
        )

        logger.debug(
            f"{log_tag_const.CSV_HANDLE} Finish manipulating text in csv file."
        )

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CSV_HANDLE} There is a error when mainpulate the text ",
                    f"in a csv file. \n{traceback.format_exc()}",
                ]
            )
        )
        return {"status": 400, "message": "", "data": ""}


def _data_clean(support_type, data, file_name):
    """Clean the data.

    support_type: support type;
    data: text content;
    """
    logger.debug(f"{log_tag_const.CSV_HANDLE} Start to clean data in csv.")

    # 去除不可见字符
    if "remove_invisible_characters" in support_type:
        clean_data = []
        for item in data:
            result = clean_transform.remove_invisible_characters({"text": item})

            if result["status"] != 200:
                return {"status": 400, "message": "去除不可见字符失败", "data": ""}

            clean_data.append(result["data"])

        data = clean_data
        data.insert(0, ["prompt"])

        # 将文件存为middle
        file_name = file_utils.get_file_name(
            {"file_name": file_name, "handle_name": "middle"}
        )

        csv_utils.save_csv(
            {"file_name": file_name, "phase_value": "middle", "data": data}
        )

    logger.debug(f"{log_tag_const.CSV_HANDLE} Finish cleaning data in csv.")

    return {"status": 200, "message": "", "data": data}
