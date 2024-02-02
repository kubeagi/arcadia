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
import traceback

from database_operate import dp_document_qa_remove_duplicate_db_operate
from utils import date_time_utils

logger = logging.getLogger(__name__)

class QARemoveDuplicate():

    def __init__(
        self,
        embeddings,
        pool
    ):
        """QA Remove Duplicated

        Args:
            embeddings (Embeddings): QA embeddings Client.
        """
        self.embeddings = embeddings
        self.pool = pool

    def _import_qa_embedding_data(
        self,
        qa_pairs
    ):
        """Vectorizing QA data and importing it into a database

        Args:
            embeddings (Embeddings): Embeddings Client
            qa_pairs: QA datasets
        """
        logger.debug(f"Starting to QA vectorize: {qa_pairs}")

        try:
            texts = []
            for qa in qa_pairs:
                texts.append(qa["question"])
                texts.append(qa["answer"])
            embeddings = self.embeddings.embed_documents(texts)
            logger.debug(f"completed QA vectorize")
            for index, qa_pair in enumerate(qa_pairs):
                create_datetime = date_time_utils.now_str()
                params = {
                    "id": qa_pair.get("id"),
                    "task_id": qa_pair.get("task_id"),
                    "document_id": qa_pair.get("document_id"),
                    "document_chunk_id": qa_pair.get("document_chunk_id"),
                    "file_name": qa_pair.get("file_name"),
                    "question": qa_pair.get("question"),
                    "question_vector": embeddings.data[index * 2].embedding,
                    "answer": qa_pair.get("answer"),
                    "answer_vector": embeddings.data[index * 2 + 1].embedding,
                    "create_datetime": create_datetime
                }
                dp_document_qa_remove_duplicate_db_operate.add(
                    params,
                    self.pool
                )

            return {"status": 200, "message": "", "data": ""}
        except Exception as ex:
            logger.error(
                "".join(
                    [
                        f"qa embedding fail\n",
                        f"The tracing error is: \n{traceback.format_exc()}\n",
                    ]
                )
            )
            return {"status": 1000, "message": "QA数据向量化失败，请检查向量化模型是否正常！", "data": ""}

    def _remove_qa_embedding_data_by_id(
        self,
        id
    ):
        """Remove QA data by ID
        """
        params = {
            "id": id
        }
        dp_document_qa_remove_duplicate_db_operate.delete_by_id(
            params,
            self.pool
        )

    def _remove_qa_embedding_data_by_document(
        self,
        document_id,
        task_id
    ):
        """Remove QA data by Document ID and Task ID
        """
        params = {
            "document_id": document_id,
            "task_id": task_id
        }
        dp_document_qa_remove_duplicate_db_operate.delete_by_task_document(
            params,
            self.pool
        )

    def _remove_duplicate_qa_data(
        self,
        qa_pairs,
        distance
    ):
        """Remove semantically similar QA based on the similarity threshold

        Args:
            qa_pairs (list): QA datasets
            distance (float): similarity threshold
        """
        try:
            qa_pairs_dict = {}
            for qa in qa_pairs:
                qa_pairs_dict[qa["id"]] = qa
            for id, qa_pair in qa_pairs_dict.items():
                logger.debug(f"Querying similarity of QA item: {qa_pair}")
                if qa_pair.get("duplicated_flag") is not None and qa_pair.get("duplicated_flag"):
                    logger.debug(f"QA Duplicate Skip")
                    continue
                params = {
                    "task_id": qa_pair.get("task_id"),
                    "document_id": qa_pair.get("document_id"),
                    "id": qa_pair.get("id"),
                }
                res = dp_document_qa_remove_duplicate_db_operate.filter_by_distance(
                    params,
                    self.pool
                )
                self._remove_qa_embedding_data_by_id(
                    qa_pair["id"]
                )
                logger.debug(f"Querying similarity of QA result: {res}")
                for qa in res["data"]:
                    if qa["question_distance"] > distance and qa["answer_distance"] > distance:
                        qa["duplicated_flag"] = 0
                        qa["compare_with_id"] = id
                        qa_pairs_dict[qa["id"]] = qa
                        self._remove_qa_embedding_data_by_id(
                            qa["id"]
                        )
            return {"status": 200, "message": "", "data": list(qa_pairs_dict.values())}
        except Exception as ex:
            logger.error(
                "".join(
                    [
                        f"qa remove duplicate fail\n",
                        f"The tracing error is: \n{traceback.format_exc()}\n",
                    ]
                )
            )
            return {"status": 1000, "message": "QA去重失败，未知原因，请联系管理员！", "data": ""}

    def qa_remove_duplicate(
        self,
        qa_pairs,
        distance
    ):
        """Vectorize QA data and remove semantically similar QA based on a similarity threshold
        
        Args:
            qa_pairs (list): QA datasets
            distance (float): similarity threshold
            
        Example:
            "qa_pairs": [
                {
                   "id": "01",
                   "document_id": "100",
                   "task_id": "200",
                   "question": "哈密瓜主要产地",
                   "answer": "哈密瓜主要产自新疆、甘肃等地"
               },
               {
                   "id": "02",
                   "document_id": "100",
                   "task_id": "200",
                   "question": "哈密瓜主要来源地",
                   "answer": "哈密瓜的主要产地包括新疆和甘肃"
               },
                {
                   "id": "03",
                   "document_id": "100",
                   "task_id": "200",
                   "question": "哈密瓜主要产地在哪里",
                   "answer": "哈密瓜主产于吐哈盆地（即吐鲁番盆地和哈密盆地的统称），它形态各异，风味独特，瓜肉肥厚，清脆爽口。哈密瓜营养丰富，含糖量最高达21%。哈密的甜瓜在东汉永平年间就成为进贡的异瓜种了。至清代，被哈密王作为贡品，受康熙赏赐而得名哈密瓜。时哈密瓜“往年进贡”、“瓜莫盛于哈密”、“瓜则充贡品者真出哈密”。追根溯源，哈密瓜却源于吐鲁番鄯善县一带"
               }
            ]
            "distance": 0.9
        
        Returns:
            [
              {
                "id": "01",
                "document_id": "100",
                "task_id": "200",
                "question": "哈密瓜主要产地",
                "answer": "哈密瓜主要产自新疆、甘肃等地"
              },
              {
                "id": "02",
                "task_id": "200",
                "document_id": "100",
                "question": "哈密瓜主要来源地",
                "answer": "哈密瓜的主要产地包括新疆和甘肃",
                "question_distance": 0.9919387102127075,
                "answer_distance": 0.9851632118225098,
                "duplicated_flag": true
              },
              {
                "id": "03",
                "document_id": "100",
                "task_id": "200",
                "question": "哈密瓜主要产地在哪里",
                "answer": "哈密瓜主产于吐哈盆地（即吐鲁番盆地和哈密盆地的统称），它形态各异，风味独特，瓜肉肥厚，清脆爽口。哈密瓜营养丰富，含糖量最高达21%。哈密的甜瓜在东汉永平年间就成为进贡的异瓜种了。至清代，被哈密王作为贡品，受康熙赏赐而得名哈密瓜。时哈密瓜“往年进贡”、“瓜莫盛于哈密”、“瓜则充贡品者真出哈密”。追根溯源，哈密瓜却源于吐鲁番鄯善县一带"
              }
            ]
        """
        qa_embedding_res = self._import_qa_embedding_data(
            qa_pairs
        )

        if qa_embedding_res.get("status") != 200:
            return qa_embedding_res

        return self._remove_duplicate_qa_data(
            qa_pairs,
            distance
        )
