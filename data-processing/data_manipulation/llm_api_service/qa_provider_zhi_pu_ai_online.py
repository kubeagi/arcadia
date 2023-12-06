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

import zhipuai
from common import log_tag_const
from common.config import config
from llm_prompt_template import zhi_pu_ai_prompt

from .base_qa_provider import BaseQAProvider

logger = logging.getLogger(__name__)


class QAProviderZhiPuAIOnline(BaseQAProvider):
    """The QA provider is used by zhi pu ai online."""

    def __init__(self, api_key=None):
        if api_key is None:
            api_key = config.zhipuai_api_key
        zhipuai.api_key = api_key


    def generate_qa_list(
        self, 
        text,
        prompt_template=None
    ):
        """Generate the QA list.
        
        Parameters
        ----------
        text
            use the text to generate QA list
        prompt_template
            the prompt template
        """
        print('xx', 'text', text)
        if prompt_template is None:
            prompt_template = zhi_pu_ai_prompt.get_default_prompt_template()

        content = prompt_template.format(
            text=text
        )
        
        result = []
        try:
            response = zhipuai.model_api.invoke(
                model="chatglm_6b",
                prompt=[{"role": "user", "content": content}],
                top_p=0.7,
                temperature=0.9,
            )
            if response['success']:
                result = self.__format_response_to_qa_list(response)
            else:
                logger.error(''.join([
                    f"{log_tag_const.ZHI_PU_AI} Cannot access the ZhiPuAI service.\n",
                    f"The error is: \n{response['msg']}\n"
                ]))
        except Exception as ex:
            result = []
            logger.error(''.join([
                f"{log_tag_const.ZHI_PU_AI} Cannot access the ZhiPuAI service.\n",
                f"The tracing error is: \n{traceback.format_exc()}\n"
            ]))

        return result


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
    