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

import asyncio
import logging
import threading

from common import log_tag_const

logger = logging.getLogger(__name__)


def run_async_background_task(task_creator, task_name):
    """Run a async background task with a new thread.

    task_creator: a function to run the background task;
    task_name: the task name which is use to identify the different task;
    """
    loop = asyncio.new_event_loop()

    thread = threading.Thread(target=task_creator, args=(loop,), name=task_name)
    thread.start()

    logger.debug(
        "".join(
            [
                f"{log_tag_const.THREADING} Start a new thread.\n",
                f"thread name: {task_name}\n",
                f"thread id: {thread.ident}",
            ]
        )
    )
