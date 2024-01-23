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

import datetime
import logging
import os
from logging.handlers import TimedRotatingFileHandler


def init_config(source_type, log_dir):
    """Initialize the log config"""
    # Disable debug logs for the Kubernetes Python client
    logging.getLogger("kubernetes").setLevel(logging.WARNING)

    os.makedirs(log_dir, exist_ok=True)
    ###
    # 配置全局日志配置
    ###
    file_handler = TimedRotatingFileHandler(
        f'{log_dir}/{source_type}_{datetime.datetime.now().strftime("%Y-%m-%d")}.log',
        when="midnight",
        interval=1,
        backupCount=30,
    )  # 按天生成日志文件，最多保存30天的日志文件

    file_handler.setLevel(logging.DEBUG)

    # 将error和critical级别的日志单独存放
    error_file_handler = TimedRotatingFileHandler(
        f'log/{source_type}_{datetime.datetime.now().strftime("%Y-%m-%d")}.err.log',
        when="midnight",
        interval=1,
        backupCount=30,
    )  # 按天生成日志文件，最多保存30天的日志文件

    error_file_handler.suffix = "%Y-%m-%d"  # 文件名的时间格式
    error_file_handler.setLevel(logging.ERROR)

    logging.basicConfig(
        level=logging.DEBUG,
        format="%(asctime)s [%(levelname)s] - %(message)s",
        handlers=[file_handler, error_file_handler, logging.StreamHandler()],
    )
