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

import ulid

from common import log_tag_const
from common.config import config
from database_operate import data_process_document_chunk_db_operate
from document_chunks.spacy_text_splitter import SpacyTextSplitter
from document_loaders.pdf import PDFLoader
from file_handle import common_handle
from file_handle.base import BaseHandle
from utils import file_utils, json_utils

logger = logging.getLogger(__name__)


class PDFHandle(BaseHandle):
    def __init__(
        self,
        file_name,
        document_id,
        support_type,
        conn_pool,
        task_id,
        create_user,
        chunk_size=None,
        chunk_overlap=None,
    ):
        """
        Initialize the pdf handle.

        Args:
            file_name: file name.
            document_id: document id.
            support_type: data processing support config type.
            conn_pool: PostgreSQL connect pool.
            task_id: data processing task id.
            create_user: create user.
            chunk_size: chunk size.
            chunk_overlap: chunk overlap.
        """
        if chunk_size is None:
            chunk_size = config.knowledge_chunk_size

        if chunk_overlap is None:
            chunk_overlap = config.knowledge_chunk_overlap
        self._file_name = file_name
        self._document_id = document_id
        self._support_type = support_type
        self._conn_pool = conn_pool
        self._task_id = task_id
        self._create_user = create_user
        self._chunk_size = chunk_size
        self._chunk_overlap = chunk_overlap

    def handle(
        self,
    ):
        """handle the text content from a pdf file."""

        logger.debug(f"{log_tag_const.PDF_HANDLE} Start to handle the text in pdf")

        try:
            pdf_file_path = file_utils.get_temp_file_path()
            file_path = pdf_file_path + "original/" + self._file_name

            # Text splitter
            documents = self._get_documents(file_path=file_path)

            # step 2
            # save all chunk info to database
            all_document_for_process = []
            for document in documents:
                chunck_id = ulid.ulid()
                content = document.page_content.replace("\n", "")
                chunk_insert_item = {
                    "id": chunck_id,
                    "document_id": self._document_id,
                    "task_id": self._task_id,
                    "status": "not_start",
                    "content": content,
                    "meta_info": json_utils.dumps(document.metadata),
                    "page_number": document.metadata.get("page") + 1,
                    "creator": self._create_user,
                }
                all_document_for_process.append(chunk_insert_item)

                data_process_document_chunk_db_operate.add(
                    chunk_insert_item, pool=self._conn_pool
                )

            response = common_handle.text_manipulate(
                file_name=self._file_name,
                all_document_for_process=all_document_for_process,
                support_type=self._support_type,
                conn_pool=self._conn_pool,
                create_user=self._create_user,
            )

            return response
        except Exception as ex:
            logger.error(
                "".join(
                    [
                        f"{log_tag_const.PDF_HANDLE} There is an error when manipulate ",
                        f"the text in pdf handler. \n{traceback.format_exc()}",
                    ]
                )
            )
            logger.debug(f"{log_tag_const.PDF_HANDLE} Finish manipulating the text in pdf")
            return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


    def _get_documents(self, file_path):
        pdf_loader = PDFLoader(file_path)
        docs = pdf_loader.load()

        text_splitter = SpacyTextSplitter(
            separator="\n\n",
            pipeline="zh_core_web_sm",
            chunk_size=int(self._chunk_size),
            chunk_overlap=int(self._chunk_overlap),
        )
        documents = text_splitter.split_documents(docs)

        return documents
