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

from sanic.handlers import ErrorHandler
from sanic.response import json

from common import log_tag_const

logger = logging.getLogger(__name__)


class CustomErrorHandler(ErrorHandler):
    """Custom the error handler for the sanic app"""

    def default(self, request, exception):
        status_code = getattr(exception, "status_code", 500)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.WEB_SERVER_ERROR} The url has a error.\n",
                    f"url: {request.url}\n",
                    f"status code: {status_code} \n",
                    f"error trace: \n{traceback.format_exc()}",
                ]
            )
        )
        return json(
            {
                "status": status_code,
                "message": str(exception),
                "data": traceback.format_exc(),
            }
        )
