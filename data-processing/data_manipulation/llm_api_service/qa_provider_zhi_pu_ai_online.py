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
import re
import traceback
import time

import zhipuai
from common import log_tag_const
from common.config import config
from llm_prompt_template import llm_prompt

from .base_qa_provider import BaseQAProvider

logger = logging.getLogger(__name__)


class QAProviderZhiPuAIOnline(BaseQAProvider):
    """The QA provider is used by zhi pu ai online."""

    def __init__(self, api_key=None):
        zhipuai.api_key = api_key


    def generate_qa_list(
        self, 
        text,
        model,
        prompt_template=None,
        top_p=None,
        temperature=None
    ):
        """Generate the QA list.
        
        Parameters
        ----------
        text
            use the text to generate QA list
        prompt_template
            the prompt template
        """
        if prompt_template is None:
            prompt_template = llm_prompt.get_default_prompt_template()
        if top_p is None:
            top_p = 0.7
        if temperature is None:
            temperature = 0.8

        content = prompt_template.format(
            text=text
        )
        
        result = []
        status = 200
        message = ''
        invoke_count = 0
        while True:
            logger.debug(''.join([
                f"{log_tag_const.ZHI_PU_AI} content.\n",
                f"{content}\n"
            ]))
            try:
                if invoke_count >= int(config.llm_qa_retry_count):
                    logger.error(''.join([
                        f"{log_tag_const.ZHI_PU_AI} Cannot access the open ai service.\n",
                        f"The tracing error is: \n{traceback.format_exc()}\n"
                    ]))

                    status = 1000
                    message = traceback.format_exc()

                    break
                else:
                    response = zhipuai.model_api.invoke(
                        model="chatglm_6b",
                        prompt=[{"role": "user", "content": content}],
                        top_p=top_p,
                        temperature=temperature,
                    )
                    if response['success']:
                        result = self.__format_response_to_qa_list(response)
                        if len(result) > 0:
                            break
                        elif invoke_count > int(config.llm_qa_retry_count):
                            logger.error(''.join([
                                f"{log_tag_const.ZHI_PU_AI} Cannot access the open ai service.\n",
                                f"The tracing error is: \n{traceback.format_exc()}\n"
                            ]))

                            status = 1000
                            message = traceback.format_exc()

                            break
                        else:
                            logger.warn('failed to get QA list, wait for 10 seconds and retry')
                            time.sleep(10) # sleep 10 seconds
                            invoke_count += 1
                    else:
                        logger.error(''.join([
                            f"{log_tag_const.ZHI_PU_AI} Cannot access the ZhiPuAI service.\n",
                            f"The error is: \n{response['msg']}\n"
                        ]))
                        logger.warn('zhipuai request failed, wait for 10 seconds and retry')
                        time.sleep(10) # sleep 10 seconds
                        invoke_count += 1
            except Exception as ex:
                logger.warn('zhipuai request exception, wait for 10 seconds and retry')
                time.sleep(10)
                invoke_count += 1

        return {
            'status': status,
            'message': message,
            'data': result
        }


    def __format_response_to_qa_list(self, response):
        """Format the response to the QA list."""
        text = response['data']['choices'][0]['content']

        pattern = re.compile(r'Q\d+:(\s*)(.*?)(\s*)A\d+:(\s*)([\s\S]*?)(?=Q|$)')
        # 移除换行符
        text = text.replace('\\n', '')
        matches = pattern.findall(text)

        result = []
        for match in matches:
            q = match[1]
            a = match[4]
            if q and a:
                result.append([q, a])

        return result
    