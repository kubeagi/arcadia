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

import ujson
import ulid
from langchain.document_loaders import PyPDFLoader
from langchain.text_splitter import SpacyTextSplitter

from common import log_tag_const
from common.config import config
from database_operate import data_process_document_chunk_db_operate
from file_handle import common_handle
from utils import file_utils

logger = logging.getLogger(__name__)


def text_manipulate(
    file_name,
    document_id,
    support_type,
    conn_pool,
    task_id,
    create_user,
    chunk_size=None,
    chunk_overlap=None
):
    """Manipulate the text content from a pdf file.
    
    file_name: file name;
    support_type: support type;
    conn_pool: database connection pool;
    task_id: data process task id;
    chunk_size: chunk size;
    chunk_overlap: chunk overlap;
    """
    
    logger.debug(f"{log_tag_const.PDF_HANDLE} Start to manipulate the text in pdf")

    try:
        pdf_file_path = file_utils.get_temp_file_path()
        file_path = pdf_file_path + 'original/' + file_name
        
        # Text splitter
        documents = _get_documents_by_langchain(
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
            file_path=file_path
        )

        # step 2
        # save all chunk info to database
        all_document_for_process = []
        for document in documents:
            chunck_id = ulid.ulid()
            page = document.metadata.get('page') + 1
            content = document.page_content.replace("\n", "")
            meta_info = document.metadata
            meta_info['source'] = file_name
            chunk_insert_item = {
                'id': chunck_id,
                'document_id': document_id,
                'task_id': task_id,
                'status': 'not_start',
                'content': content,
                'meta_info': ujson.dumps(meta_info, ensure_ascii=False),
                'page_number': page,
                'creator': create_user
            }
            all_document_for_process.append(chunk_insert_item)

            data_process_document_chunk_db_operate.add(
                chunk_insert_item,
                pool=conn_pool
            )

        response = common_handle.text_manipulate(
            file_name=file_name,
            all_document_for_process=all_document_for_process,
            support_type=support_type,
            conn_pool=conn_pool,
            create_user=create_user
        )

        return response
    except Exception as ex:
        logger.error(''.join([
            f"{log_tag_const.PDF_HANDLE} There is an error when manipulate ",
            f"the text in pdf handler. \n{traceback.format_exc()}"
        ]))
        logger.debug(f"{log_tag_const.PDF_HANDLE} Finish manipulating the text in pdf")
        return {
            'status': 400,
            'message': str(ex),
            'data': traceback.format_exc()
        }

def _get_documents_by_langchain(
    chunk_size,
    chunk_overlap,
    file_path
):
    # Split the text.
    if chunk_size is None:
        chunk_size = config.knowledge_chunk_size

    if chunk_overlap is None:
        chunk_overlap = config.knowledge_chunk_overlap

    source_reader = PyPDFLoader(file_path)
    text_splitter = SpacyTextSplitter(
        separator="\n\n",
        pipeline="zh_core_web_sm",
        chunk_size=int(chunk_size),
        chunk_overlap=int(chunk_overlap)
    )
    documents = source_reader.load_and_split(text_splitter)

    return documents
