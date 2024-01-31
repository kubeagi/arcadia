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

from utils import log_utils,json_utils
from embeddings.openai_embeddings import OpenAIEmbeddings
from service.data_process_qa_remove_duplicate import QARemoveDuplicate
from database_clients import postgresql_pool_client

import logging
import psycopg2

logger = logging.getLogger(__name__)

###
# 初始化日志配置
###
log_utils.init_config(
    source_type='manipulate_server',
    log_dir="log"
)

if __name__ == '__main__':
    
    qa_pairs = [
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
        },
        {
            "id": "04",
            "document_id": "101",
            "task_id": "200",
            "question": "哈密瓜主要产地",
            "answer": "哈密瓜主要产自新疆、甘肃等地"
        },
        {
            "id": "05",
            "document_id": "101",
            "task_id": "200",
            "question": "哈密瓜主要来源地",
            "answer": "哈密瓜的主要产地包括新疆和甘肃"
        },
        {
            "id": "06",
            "document_id": "101",
            "task_id": "200",
            "question": "哈密瓜主要产地在哪里",
            "answer": "哈密瓜主产于吐哈盆地（即吐鲁番盆地和哈密盆地的统称），它形态各异，风味独特，瓜肉肥厚，清脆爽口。哈密瓜营养丰富，含糖量最高达21%。哈密的甜瓜在东汉永平年间就成为进贡的异瓜种了。至清代，被哈密王作为贡品，受康熙赏赐而得名哈密瓜。时哈密瓜“往年进贡”、“瓜莫盛于哈密”、“瓜则充贡品者真出哈密”。追根溯源，哈密瓜却源于吐鲁番鄯善县一带"
        }
    ]
    base_url = "http://fastchat-api.172.22.96.167.nip.io/v1"
    api_key = "fake"
    model = "ccb8e2eb-26f1-43b4-9bcf-b88d0c359992"
    distance = 0.9
    
    def _create_database_connection():
        """Create a database connection."""
        return psycopg2.connect(
                    host="192.168.203.136", 
                    port=15432,
                    user="postgres",
                    password="123456", 
                    database="postgres"
                )

    conn_pool = postgresql_pool_client.get_pool(_create_database_connection)

    embeddings = OpenAIEmbeddings(
        base_url,
        api_key,
        model,
    )
    qa = QARemoveDuplicate(
        embeddings,
        conn_pool
        ).qa_remove_duplicate(qa_pairs, distance)
    logger.debug(json_utils.dumps(qa))
    
    postgresql_pool_client.release_pool(conn_pool)

    """ QA Remove Duplicate Result
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
            "document_id": "100",
            "task_id": "200",
            "question": "哈密瓜主要来源地",
            "answer": "哈密瓜的主要产地包括新疆和甘肃",
            "question_distance": 0.9685559272766113,
            "answer_distance": 0.9504443407058716,
            "duplicated_flag": true
        },
        {
           "id": "03",
           "document_id": "100",
            "task_id": "200",
            "question": "哈密瓜主要产地在哪里",
            "answer": "哈密瓜主产于吐哈盆地（即吐鲁番盆地和哈密盆地的统称），它形态各异，风味独特，瓜肉肥厚，清脆爽口。哈密瓜营养丰富，含糖量最高达21%。哈密的甜瓜在东汉永平年间就成为进贡的异瓜种了。至清代，被哈密王作为贡品，受康熙赏赐而得名哈密瓜。时哈密瓜“往年进贡”、“瓜莫盛于哈密”、“瓜则充贡品者真出哈密”。追根溯源，哈密瓜却源于吐鲁番鄯善县一带"
        },
        {
            "id": "04",
            "document_id": "101",
            "task_id": "200",
            "question": "哈密瓜主要产地",
            "answer": "哈密瓜主要产自新疆、甘肃等地"
        },
        {
            "id": "05",
            "document_id": "101",
            "task_id": "200",
            "question": "哈密瓜主要来源地",
            "answer": "哈密瓜的主要产地包括新疆和甘肃",
            "question_distance": 0.9685559272766113,
            "answer_distance": 0.9504443407058716,
            "duplicated_flag": true
        },
        {
            "id": "06",
            "document_id": "101",
            "task_id": "200",
            "question": "哈密瓜主要产地在哪里",
            "answer": "哈密瓜主产于吐哈盆地（即吐鲁番盆地和哈密盆地的统称），它形态各异，风味独特，瓜肉肥厚，清脆爽口。哈密瓜营养丰富，含糖量最高达21%。哈密的甜瓜在东汉永平年间就成为进贡的异瓜种了。至清代，被哈密王作为贡品，受康熙赏赐而得名哈密瓜。时哈密瓜“往年进贡”、“瓜莫盛于哈密”、“瓜则充贡品者真出哈密”。追根溯源，哈密瓜却源于吐鲁番鄯善县一带"
        }
    ]
    """
