# Copyright 2024 KubeAGI.
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

import os

from langchain_openai import OpenAIEmbeddings as BaseOpenAIEmbeddings
from ragas.embeddings import BaseRagasEmbeddings

NO_KEY = "NO_KEY"

# DEPRECATED: using ragas.embeddings.LangchainEmbeddingsWrapper instead

class OpenAIEmbeddings(BaseOpenAIEmbeddings, BaseRagasEmbeddings):
    api_key: str = NO_KEY

    def __init__(
        self, api_key: str = NO_KEY, api_base: str = NO_KEY, model_name: str = NO_KEY
    ):
        # api key
        key_from_env = os.getenv("OPENAI_API_KEY", NO_KEY)
        if key_from_env != NO_KEY:
            openai_api_key = key_from_env
        else:
            openai_api_key = api_key
        super(BaseOpenAIEmbeddings, self).__init__(
            openai_api_key=openai_api_key, openai_api_base=api_base, model=model_name
        )
        self.api_key = openai_api_key

    def validate_api_key(self):
        if self.openai_api_key == NO_KEY:
            os_env_key = os.getenv("OPENAI_API_KEY", NO_KEY)
            if os_env_key != NO_KEY:
                self.api_key = os_env_key
            else:
                raise ValueError("openai api key must be provided")
