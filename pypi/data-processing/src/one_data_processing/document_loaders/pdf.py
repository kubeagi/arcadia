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

from langchain.document_loaders import PyPDFLoader
from langchain_core.documents import Document

from common import log_tag_const
from document_loaders.base import BaseLoader
from utils import file_utils

logger = logging.getLogger(__name__)

class PDFLoader(BaseLoader):
    """Load pdf file."""

    def __init__(
        self,
        file_path: str,
    ):
        """
        Initialize the loader with a list of URL paths.

        Args:
            file_path (str): File Path.
        """
        self._file_path = file_path

    def load(self) -> List[Document]:
        """
        Load and return all Documents from the docx file.

        Returns:
            List[Document]: A list of Document objects.

        """
        logger.info(f"{log_tag_const.PDF_LOADER} Start to load pdf file")

        # Get file name
        file_name = file_utils.get_file_name(self._file_path)

        pdf_loader = PyPDFLoader(self._file_path)
        documents = pdf_loader.load()
        for document in documents:
            document.metadata["source"] = file_name

        return documents
