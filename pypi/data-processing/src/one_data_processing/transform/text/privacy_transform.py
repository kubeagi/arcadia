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

from common import log_tag_const

logger = logging.getLogger(__name__)


def remove_email(text, replace_string=None):
    """Replace email info with the user defined string.

    text: text;
    replace_string: the text is used to replace the email info;

    usage:
    input:
    如果需要可以联系官方邮箱:172817631@qq.com马上申请为你开通

    output:
    如果需要可以联系官方邮箱:xxxxxx马上申请为你开通
    """
    try:
        if replace_string is None:
            replace_string = "xxxxxx"

        pattern = r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"
        find_pattern = (
            r"[^，。！？,.!?]*[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}[^，。！？,.!?]*"
        )

        clean_text = re.sub(pattern, replace_string, text)

        clean_data = _find_clean_data(
            text=text,
            pattern=pattern,
            find_pattern=find_pattern,
            replace_string=replace_string,
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
                    f"{log_tag_const.CLEAN_TRANSFORM} Execute removing email.\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )
        return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


def remove_ip_address(text, replace_string=None):
    """the ip addresses are replaced with xxxxxx.

    text: text;
    replace_string: the text is used to replace the email info;

    usage:
    input:
    服务器登陆ip为192.168.255.255

    output:
    服务器登陆ip为xxxxxx
    """
    try:
        if replace_string is None:
            replace_string = "xxxxxx"

        pattern = "".join(
            [
                r"((?:(?:1[0-9][0-9]\.)|(?:2[0-4][0-9]\.)|",
                r"(?:25[0-5]\.)|(?:[1-9][0-9]\.)|(?:[0-9]\.))",
                r"{3}(?:(?:1[0-9][0-9])|(?:2[0-4][0-9])|",
                r"(?:25[0-5])|(?:[1-9][0-9])|(?:[0-9]))|",
                r"([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4})",
            ]
        )

        find_pattern = "".join([r"([^，。！？,.!?]*)", pattern, r"([^，。！？,.!?]*)"])

        clean_text = re.sub(pattern=pattern, repl=replace_string, string=text)

        clean_data = []
        sentences = re.findall(find_pattern, text)
        for sentence in sentences:
            sentence = "".join([sentence[0], sentence[1], sentence[3]])
            post_content = re.sub(pattern, replace_string, sentence)
            clean_data.append({"pre_content": sentence, "post_content": post_content})

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
                    f"{log_tag_const.PRIVACY_TRANSFORM} Executing remove email failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def remove_phone(text, replace_string=None):
    """the phone are replaced with xxxxxx.

    text: text;
    replace_string: the text is used to replace the email info;

    usage:
    input:
    12345678910, 我的手机号是: 18617261536,我的座机号是: 029-1234567

    output:
    12345678910, 我的手机号是: xxxxxx,我的座机号是: 029-1234567
    """
    try:
        if replace_string is None:
            replace_string = "xxxxxx"

        pattern = r"((\+|00)86)?(1)((3[\d])|(4[5,6,7,9])|(5[0-3,5-9])|(6[5-7])|(7[0-8])|(8[\d])|(9[1,8,9]))(\d{8})(?![0-9])"
        find_pattern = "".join([r"([^，。！？,.!?]*)", pattern, r"([^，。！？,.!?]*)"])

        clean_text = re.sub(pattern=pattern, repl=replace_string, string=text)

        clean_data = []
        sentences = re.findall(find_pattern, text)
        for sentence in sentences:
            sentence = "".join(
                [sentence[0], sentence[3], sentence[4], sentence[12], sentence[13]]
            )
            post_content = re.sub(pattern, replace_string, sentence)
            clean_data.append({"pre_content": sentence, "post_content": post_content})

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
                    f"{log_tag_const.PRIVACY_TRANSFORM} Executing remove phone failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def remove_id_card(text, replace_string=None):
    """the phone are replaced with xxxxxx.

    text: text;
    replace_string: the text is used to replace the email info;

    usage:
    input:
    身份证号1：123451230112121234，身份证号2：12345123011212123x，位身份证号3：123456780009876

    output:
    身份证号1：xxxxxx，身份证号2：xxxxxx，位身份证号3：xxxxxx
    """
    try:
        if replace_string is None:
            replace_string = "xxxxxx"

        id_card_regex = [
            r"\b([1-9]\d{5}[1-9]\d{3})((0\d)|(1[0-2]))(([0|1|2]\d)|(3[0-1]))(\d{3}[0-9Xx])(?![0-9])",
            r"\b([1-9]\d{7})((0\d)|(1[0-2]))(([0-2][1-9])|(3[0-1]))(\d{2}[0-9Xx])(?![0-9])",
        ]

        clean_data = []
        for regex_exp in id_card_regex:
            find_pattern = "".join([r"([^，。！？,.!?]*)", regex_exp, r"([^，。！？,.!?]*)"])

            sentences = re.findall(find_pattern, text)

            text = re.sub(pattern=regex_exp, repl=replace_string, string=text)

            for sentence in sentences:
                sentence = "".join(
                    [
                        sentence[0],
                        sentence[1],
                        sentence[2],
                        sentence[5],
                        sentence[8],
                        sentence[9],
                    ]
                )
                post_content = re.sub(regex_exp, replace_string, sentence)
                clean_data.append(
                    {"pre_content": sentence, "post_content": post_content}
                )

        return {
            "status": 200,
            "message": "",
            "data": {"clean_data": clean_data, "text": text},
        }

    except Exception as ex:
        error = str(ex)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.PRIVACY_TRANSFORM} Executing remove id card failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def remove_weixin(text, replace_string=None):
    """the weixin are replaced with xxxxxx.

    text: text;
    replace_string: the text is used to replace the email info;

    usage:
    input:
    我的微信号：qw123456

    output:
    我的xxxxxx
    """
    try:
        if replace_string is None:
            replace_string = "xxxxxx"
        weixin_regex = [
            r"vxin[：|:][a-zA-Z0-9{3,20}]+",
            r"vx[：|:][a-zA-Z0-9{3,20}]+",
            r"VX[：|:][a-zA-Z0-9{3,20}]+",
            r"Vxin[：|:][a-zA-Z0-9{3,20}]+",
            r"wx[：|:][a-zA-Z0-9{3,20}]+",
            r"WX[：|:][a-zA-Z0-9{3,20}]+",
            r"wei xin[：|:][a-zA-Z0-9{3,20}]+",
            r"weixin[：|:][a-zA-Z0-9{3,20}]+",
            r"微信[：|:][a-zA-Z0-9{3,20}]+",
            r"微信号[：|:][a-zA-Z0-9{3,20}]+",
            r"薇信[：|:][a-zA-Z0-9{3,20}]+",
            r"薇信号[：|:][a-zA-Z0-9{3,20}]+",
            r"v信[：|:][a-zA-Z0-9{3,20}]+",
            r"V信[：|:][a-zA-Z0-9{3,20}]+",
        ]

        clean_data = []
        for regex_exp in weixin_regex:
            find_pattern = "".join([r"[^，。！？,.!?]*", regex_exp, r"[^，。！？,.!?]*"])
            sentences = re.findall(find_pattern, text)

            text = re.sub(pattern=regex_exp, repl=replace_string, string=text)

            for sentence in sentences:
                post_content = re.sub(regex_exp, replace_string, sentence)
                clean_data.append(
                    {"pre_content": sentence, "post_content": post_content}
                )

        return {
            "status": 200,
            "message": "",
            "data": {"clean_data": clean_data, "text": text},
        }

    except Exception as ex:
        error = str(ex)
        logger.error(
            "".join(
                [
                    f"{log_tag_const.PRIVACY_TRANSFORM} Executing remove id card failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def remove_bank_card(text, replace_string=None):
    """the remove bank card are replaced with xxxxxx.

    text: text;

    usage:
    input:
    银行卡号1：1234567890123456，银行卡号2：12345678901234567，银行卡号3：1234567890123456789

    output:
    银行卡号1：xxxxxx，银行卡号2：12345678901234567，银行卡号3：xxxxxx
    """
    try:
        if replace_string is None:
            replace_string = "xxxxxx"

        pattern = r"\b([1-9]{1})(\d{15}|\d{18})(?![0-9])"
        find_pattern = (
            r"([^，。！？,.!?]*)\b([1-9]{1})(\d{15}|\d{18})((?![0-9])[^，。！？,.!?]*)"
        )

        clean_text = re.sub(pattern=pattern, repl=replace_string, string=text)

        clean_data = _find_clean_data(
            text=text,
            pattern=pattern,
            find_pattern=find_pattern,
            replace_string=replace_string,
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
                    f"{log_tag_const.PRIVACY_TRANSFORM} Executing remove email failed\n",
                    f"The tracing error is: \n{traceback.format_exc()}\n",
                ]
            )
        )

        return {"status": 400, "message": error, "data": traceback.format_exc()}


def _find_clean_data(text, replace_string, pattern, find_pattern):
    """find clean data for pre_content and post_content.

    text: text;
    pattern: ;
    find_pattern: ;
    replace_string: replace string for privacy


    """
    clean_data = []

    sentences = re.findall(find_pattern, text)
    for sentence in sentences:
        post_content = re.sub(pattern, replace_string, sentence)
        clean_data.append({"pre_content": sentence, "post_content": post_content})

    return clean_data
