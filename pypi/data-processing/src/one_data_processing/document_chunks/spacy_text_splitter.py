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

from typing import List

from langchain_core.documents import Document

from document_chunks.base import TextSplitter


class SpacyTextSplitter(TextSplitter):
    def __init__(
        self,
        separator: str = "\n\n",
        pipeline: str = "zh_core_web_sm",
        chunk_size: int = 500,
        chunk_overlap: int = 10,
    ):
        """Initialize the spacy text splitter."""
        if chunk_overlap > chunk_size:
            raise ValueError(
                f"Got a larger chunk overlap ({chunk_overlap}) than chunk size "
                f"({chunk_size}), should be smaller."
            )
        self._chunk_size = chunk_size
        self._chunk_overlap = chunk_overlap
        self._separator = separator
        self._pipeline = pipeline

    def split_documents(self, documents: List[Document]) -> List[Document]:
        from langchain.text_splitter import SpacyTextSplitter
        text_splitter = SpacyTextSplitter(
            separator=self._separator,
            pipeline=self._pipeline,
            chunk_size=self._chunk_size,
            chunk_overlap=self._chunk_overlap,
        )
        documents = text_splitter.split_documents(documents)
        return documents
