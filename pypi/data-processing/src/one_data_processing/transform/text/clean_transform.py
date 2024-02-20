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
import re
import traceback

import ftfy
import opencc
from selectolax.parser import HTMLParser

from common import log_tag_const, special_characters

logger = logging.getLogger(__name__)


def remove_invisible_characters(text):
    """remove invisible characters.

    text: text;

    usage
    input:
    “一户一表、水表出户、抄表到户”是指一个家庭用户安装一个计量水表，计量水表安装在住宅的公共部位，供水企业抄表到户，按户计量收费。
    output:
    “一户一表、水表出户、抄表到户”是指一个家庭用户安装一个计量水表，计量水表安装在住宅的公共部位，供水企业抄表到户，按户计量收费。
    """
    try:
        pattern = r"[\x00-\x1F\x7F-\x9F\xAD\r\t\b\x0B\x1C\x1D\x1E]"
        find_pattern = (
            r"[^，。！？,.!?]*[\x00-\x1F\x7F-\x9F\xAD\r\t\b\x0B\x1C\x1D\x1E][^，。！？,.!?]*"
        )
        replace_text = ""

        clean_text = re.sub(pattern, replace_text, text)

        clean_data = _find_clean_data(
            text=text,
            pattern=pattern,
            find_pattern=find_pattern,
            replace_text=replace_text,
        )
        return {
            "status": 200,
            "message": "",
            "data": {"clean_data": clean_data, "text": clean_text},
        }
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CLEAN_TRANSFORM} Execute removing invisible characters failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )
        return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


def space_standardization(text):
    """space standardization.

    text: text;

    usage:
    input:
    第一条　灭火是指国家综合性消防救援队、专职消防队依法承担的火灾扑救工作。

    output:
    第一条 灭火是指国家综合性消防救援队、专职消防队依法承担的火灾扑救工作。
    """
    try:
        various_whitespaces = special_characters.VARIOUS_WHITESPACES
        pattern = "|".join(re.escape(value) for value in various_whitespaces)
        find_pattern = "|".join(
            f"[^，。！？,.!?]*{re.escape(value)}[^，。！？,.!?]*"
            for value in various_whitespaces
        )
        replace_text = " "

        clean_text = re.sub(pattern, replace_text, text)

        clean_data = _find_clean_data(
            text=text,
            pattern=pattern,
            find_pattern=find_pattern,
            replace_text=replace_text,
        )

        return {
            "status": 200,
            "message": "",
            "data": {"clean_data": clean_data, "text": clean_text},
        }
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CLEAN_TRANSFORM} Executing space standardization failed.\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )
        return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


def remove_garbled_text(text):
    """remove garbled text.
    text: text;

    usage:
    input:
    江苏省滨海县人民法院民事判决书(2015)滨滩商初字第0014号原告孟庆连,男,49岁,居民。委托代理人王成庭,滨海县滨淮法律服务所法律工作者。 â€” like this one.

    output:
    江苏省滨海县人民法院民事判决书(2015)滨滩商初字第0014号原告孟庆连,男,49岁,居民。委托代理人王成庭,滨海县滨淮法律服务所法律工作者。 — like this one.

    """
    try:
        clean_text = ftfy.fix_text(text)
        return {"status": 200, "message": "", "data": {"found": 0, "text": clean_text}}

    except Exception as ex:
        error = str(ex)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CLEAN_TRANSFORM} Executing space standardization failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def traditional_to_simplified(text):
    """Traditional Chinese to Simplified Chinese.

    text: text;

    usage:
    input:
    風暴帶來的暫停使消防員和其他緊急反應人員得以進入禁區進行結構破壞評估。

    output:
    风暴带来的暂停使消防员和其他紧急反应人员得以进入禁区进行结构破坏评估。
    """
    try:
        clean_text = opencc.OpenCC("t2s").convert(text)

        return {"status": 200, "message": "", "data": {"found": 0, "text": clean_text}}
    except Exception as ex:
        error = str(ex)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CLEAN_TRANSFORM} Executing Traditional Chinese to Simplified Chinese failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def remove_html_tag(text):
    """clean html code in text samples.

    text: text;

    usage:
    input:
    <div class='center'><span class='bolded'>朗播 SAT 学员成绩单分析报告

    output:
    朗播 SAT 学员成绩单分析报告
    """
    try:
        text = text.replace("<li>", "\n*")
        text = text.replace("</li>", "")
        text = text.replace("<ol>", "\n*")
        text = text.replace("</ol>", "")
        parser = HTMLParser(text)

        clean_text = parser.text()

        return {"status": 200, "message": "", "data": {"found": 0, "text": clean_text}}
    except Exception as ex:
        error = str(ex)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CLEAN_TRANSFORM} Executing clean html code in text samples failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def remove_emojis(text):
    """remove emojis.

    text: text;

    usage:
    input:
    这是一段带有表情符号😊的文本。

    output:
    这是一段带有表情符号的文本。
    """
    try:
        emojis = special_characters.EMOJI
        pattern = "|".join(re.escape(value) for value in emojis)
        find_pattern = "|".join(
            f"[^，。！？,.!?]*{re.escape(value)}[^，。！？,.!?]*" for value in emojis
        )
        replace_text = ""

        clean_text = re.sub(pattern, replace_text, text)

        clean_data = _find_clean_data(
            text=text,
            pattern=pattern,
            find_pattern=find_pattern,
            replace_text=replace_text,
        )

        return {
            "status": 200,
            "message": "",
            "data": {"clean_data": clean_data, "text": clean_text},
        }

    except Exception as ex:
        error = str(ex)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.CLEAN_TRANSFORM} Executing remove emojis failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def _find_clean_data(text, pattern, find_pattern, replace_text):
    """find clean data for pre_content and post_content.

    text: text;
    pattern: ;
    find_pattern: ;

    """
    clean_data = []

    sentences = re.findall(find_pattern, text)
    for sentence in sentences:
        post_content = re.sub(pattern, replace_text, sentence)
        clean_data.append({"pre_content": sentence, "post_content": post_content})

    return clean_data
