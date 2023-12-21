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

from common import log_tag_const
from file_handle import common_handle
from utils import file_utils, pdf_utils

logger = logging.getLogger(__name__)


def text_manipulate(
    file_name,
    document_id,
    support_type,
    conn_pool,
    task_id,
    create_user,
    chunk_size,
    chunk_overlap
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
        
        # step 1
        # Get the content from the pdf fild.
        content = pdf_utils.get_content(file_path)
        logger.debug(f"{log_tag_const.PDF_HANDLE} The pdf content is\n {content}")

        response = common_handle.text_manipulate(
            file_name=file_name,
            document_id=document_id,
            content=content,
            support_type=support_type,
            conn_pool=conn_pool,
            task_id=task_id,
            create_user=create_user,
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap
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
