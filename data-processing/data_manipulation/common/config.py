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
import os
import traceback

import yaml
from utils.class_utils import Singleton

from . import log_tag_const

logger = logging.getLogger(__name__)


class Config(metaclass=Singleton):
    """Configuration class to store the env values."""
    
    def __init__(self, yaml_file_path='config.yml'):
        logger.debug(f"{log_tag_const.CONFIG} start to load config file.")
        yaml_data = self.__get_default_yaml_data()
        try: 
            with open(yaml_file_path, 'r') as file:
                # load yaml data
                yaml_data = yaml.safe_load(file)
        except Exception as ex:
            logger.error(''.join([
                f"{log_tag_const.CONFIG} There is an error when load the config "
                f"(file path =  {yaml_file_path}). \n"
                f"{traceback.format_exc()}"
            ]))
        logger.debug(''.join([
                    f"{log_tag_const.CONFIG} The content is config.\n",
                    f"{yaml_data}\n",
                ]))
        
        self.__set_property_value(yaml_data)
        


    def __get_default_yaml_data(self):
        """Get the default yaml data."""
        return {
            'minio': {},
            'zhipuai': {},
            'llm': {
                'open_ai': {}
            },
            'knowledge': {},
            'backendPg': {}
        }

    def __set_property_value(self, yaml_data):
        """设置属性的值"""
        # minio access key
        self.minio_access_key = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='minio',
            key='access_key',
            env_name='MINIO_ACCESSKEY',
            default_value='hpU4SCmj5jiU7IP5'
        )
        # minio secret key
        self.minio_secret_key = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='minio',
            key='secret_key',
            env_name='MINIO_SECRETKEY',
            default_value='7AUewBESqvKijdnNskm8nU6emTZ3rG8F'
        )
        # minio api url
        self.minio_api_url = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='minio',
            key='api_url',
            env_name='MINIO_API_URL',
            default_value='kubeagi-minio.172.22.96.136.nip.io'
        )        
        # minio secure
        # if use HTTP, secure = False; 
        # if use HTTPS, secure = True;
        self.minio_secure = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='minio',
            key='secure',
            env_name='MINIO_SECURE',
            default_value=True
        )
        # minio data set prefix
        self.minio_dataset_prefix = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='minio',
            key='dataset_prefix',
            env_name='MINIO_DATASET_PREFIX',
            default_value='dataset'
        )

        # zhi pu ai
        # api key
        self.zhipuai_api_key = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='zhipuai',
            key='api_key',
            env_name='ZHIPUAI_API_KEY',
            default_value='871772ac03fcb9db9d4ce7b1e6eea210.VZZVy0mCox0WrzQI'
        )

        # llm
        # use type such as zhipuai_online or open_ai
        self.llm_use_type = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='llm',
            key='use_type',
            env_name='LLM_USE_TYPE',
            default_value='zhipuai_online'
        )

        self.llm_qa_retry_count = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='llm',
            key='qa_retry_count',
            env_name='LLM_QA_RETRY_COUNT',
            default_value=100
        )

        # open ai
        # key
        self.open_ai_default_key = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='open_ai',
            key='key',
            env_name='OPEN_AI_DEFAULT_KEY',
            default_value='happy'
        )
        # base url
        self.open_ai_default_base_url = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='open_ai',
            key='base_url',
            env_name='OPEN_AI_DEFAULT_BASE_URL',
            default_value='http://arcadia-fastchat.172.22.96.167.nip.io/v1'
        )
        # default model
        self.open_ai_default_model = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='open_ai',
            key='model',
            env_name='OPEN_AI_DEFAULT_MODEL',
            default_value='baichuan2-7b-worker-baichuan-sample-playground'
        )

        # knowledge
        # chunk size
        self.knowledge_chunk_size = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='knowledge',
            key='chunk_size',
            env_name='KNOWLEDGE_CHUNK_SIZE',
            default_value=500
        )
        # chunk overlap
        self.knowledge_chunk_overlap = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='knowledge',
            key='chunk_overlap',
            env_name='KNOWLEDGE_CHUNK_OVERLAP',
            default_value=50
        )

        # backend PostgreSQL
        # host
        self.pg_host = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='backendPg',
            key='host',
            env_name='PG_HOST',
            default_value='localhost'
        )
        # port
        self.pg_port = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='backendPg',
            key='port',
            env_name='PG_HOST',
            default_value=5432
        )   
        # user
        self.pg_user = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='backendPg',
            key='user',
            env_name='PG_USER',
            default_value='postgres'
        )
        # password   
        self.pg_password = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='backendPg',
            key='password',
            env_name='PG_PASSWORD',
            default_value='123456'
        ) 
        # database name
        self.pg_database = self.__get_value_by_key_in_yaml(
            yaml_data,
            parent_key='backendPg',
            key='database',
            env_name='PG_DATABASE',
            default_value='arcadia'
        )        


    def __get_value_by_key_in_yaml(
        self,
        config_json, 
        parent_key,
        key,
        env_name,
        default_value
    ):
        """Get the value by key int the yaml file.
        
        Parameters
        ----------
        config_json
            the config json
        parent_key
            the parent key.
        env_name
            the environment variable name.
        default_value:
            the default value.
        """
        value = config_json[parent_key].get(key)
        if value is None:
            value = os.getenv(env_name, default_value)
        else:
            if value.startswith('${'):
                values_in_yaml = value.split(': ')
                if len(values_in_yaml) == 2:
                    env_name_in_yaml = values_in_yaml[0].strip()[2:]
                    default_value_in_yaml = values_in_yaml[1].strip()[:-1]

                    value = os.getenv(env_name_in_yaml, default_value_in_yaml)

        return value


 
config = Config()

        


        





