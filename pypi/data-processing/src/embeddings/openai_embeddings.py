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

import logging
from typing import List

from embeddings.embeddings import Embeddings

logger = logging.getLogger(__name__)

class OpenAIEmbeddings(Embeddings):

    def __init__(
        self,
        base_url: str,
        api_key: str,
        model: str,
    ):
        """OpenAI embedding models.

        Args:
            base_url (str): to support OpenAI Service custom endpoints.
            api_key (str): to support OpenAI Service API KEY.
            model (str): Embeddings Model.

        Raises:
            ImportError: If the required 'openai' package is not installed.
        """

        self.base_url = base_url
        self.api_key = api_key
        self.model = model

        try:
            from openai import OpenAI
        except ImportError:
            raise ImportError(
                "openai embedding is required for OpenAI. "
                "Please install it with `pip install openai`."
            )
        self.client = OpenAI(
            base_url=base_url,
            api_key=api_key
        )

    def embed_documents(self, texts: List[str]) -> List[List[float]]:
        """Embed search texts."""
        logger.debug(texts)
        embeddings = self.client.embeddings.create(
            model=self.model,
            input=texts
        )
        return embeddings
