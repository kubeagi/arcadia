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
import time
import traceback

from common import log_tag_const
from common.config import config
from langchain.chains import LLMChain
from langchain.llms import OpenAI
from langchain.prompts import PromptTemplate
from llm_prompt_template import open_ai_prompt

from .base_qa_provider import BaseQAProvider

logger = logging.getLogger(__name__)

class QAProviderOpenAI(BaseQAProvider):
    """The QA provider is used by open ai."""
    
    def __init__(
        self, 
        api_key=None,
        base_url=None,
        model=None
    ):
        if api_key is None:
            api_key = config.open_ai_default_key
        if base_url is None:
            base_url = config.open_ai_default_base_url
        if model is None:
            model = config.open_ai_default_model
        
        self.llm = OpenAI(
            openai_api_key=api_key, 
            base_url=base_url,
            model=model
        ) 

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
        if prompt_template is None:
            prompt_template = open_ai_prompt.get_default_prompt_template()
        
        prompt = PromptTemplate(
                    template=prompt_template, 
                    input_variables=["text"]
                )
        llm_chain = LLMChain(
            prompt=prompt, 
            llm=self.llm
        )

        result = []
        invoke_count = 0
        while True:
            try:
                response = llm_chain.run(text)
                result = self.__get_qa_list_from_response(response)
                if len(result) > 0 or invoke_count > int(config.llm_qa_retry_count):
                    logger.debug(''.join([
                        f"{log_tag_const.OPEN_AI} The QA list is \n",
                        f"\n{result}\n"
                    ]))
                    break
                else:
                    logger.warn('failed to get QA list, wait for 2 seconds and retry')
                    time.sleep(5) # sleep 5 seconds
                invoke_count += 1
            except Exception as ex:
                result = []
                logger.error(''.join([
                    f"{log_tag_const.OPEN_AI} Cannot access the open ai service.\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n"
                ]))
                time.sleep(5)
        
        return result

    
    def __get_qa_list_from_response(
        self,
        response
    ):
        """Get the QA list from the response.
        
        Notice: There are some problems in the local OpenAI service.
        Some time it cannot return the correct question and answer list.

        Parameters
        ----------
        response
            the response from open ai service
        """
        result = []

        pattern = re.compile(r'Q\d+:(\s*)(.*?)(\s*)A\d+:(\s*)([\s\S]*?)(?=Q|$)')


        # 移除换行符
        response_text = response.replace('\\n', '')
        matches = pattern.findall(response_text)

        result = []
        for match in matches:
            q = match[1]
            a = match[4]
            if q and a:
                a = re.sub(r'[\n]', '', a).strip()
                result.append([q, a])

        
        return result




    
