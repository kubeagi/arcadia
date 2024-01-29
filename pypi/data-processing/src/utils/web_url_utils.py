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

import fnmatch
import logging
import time
import traceback
from io import BytesIO
from urllib.parse import urldefrag, urljoin, urlparse

import requests
import ulid
from bs4 import BeautifulSoup
from PIL import Image
from playwright.sync_api import sync_playwright

from common import log_tag_const
from utils import date_time_utils

logger = logging.getLogger(__name__)

def handle_website(
        url,
        interval_time=1,
        resource_types=[],
        max_depth=1,
        max_count=100,
        exclude_sub_urls=[],
        include_sub_urls=[],
        exclude_img_info={"weight": 250, "height" : 250}
):
    """Recursively crawling the content, images, and other resources on a website

    Args:
        url (str): The URL to crawl.
        interval_time (int): The interval time between crawling web pages.
        resource_types (list): Crawling the types of resource on a web page.
        max_depth (int): The max depth of the recursive loading.
        max_count (int): The max web count of the recursive loading.
        exclude_sub_urls (list): Excluding subpage urls while crawling a website.
        include_sub_urls (list): Including subpage urls while crawling a website.
        exclude_img_info (dict): Excluding images with specified attributes (eg: weight、height) while crawling a website.
    Returns:
        [
            {
                "id": "9D4oAQWh",
                "pid": "0",
                "url": "https://mp.weixin.qq.com/s/65DFY...",
                "type": "page",
                "level": "1"
            }
        ]
    Examples:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
            "interval_time": 1,
            "resource_types": [
                "png",
                "jpg"
            ],
            "max_depth": 1,
            "max_count": 100,
            "exclude_sub_urls": [
                ""
            ],
            "include_sub_urls": [
                ""
            ],
            "exclude_img_info": {
                "weight": 250,
                "height": 250
            }
        }
    Raises:
    """
    logger.debug(f"{log_tag_const.WEB_CRAWLING} start time {date_time_utils.now_str()}")
    logger.debug(
        "\n".join(
            [
                f"{log_tag_const.WEB_CRAWLING} In parameter:",
                f"  url: {url}",
                f"  interval_time: {interval_time}" , 
                f"  resource_types: {resource_types} " ,
                f"  max_depth: {max_depth} ",
                f"  max_count: {max_count} ",
                f"  exclude_sub_urls: {exclude_sub_urls} ",
                f"  include_sub_urls: {include_sub_urls} ",
                f"  exclude_img_info: {exclude_img_info}"
            ]
        )
    )
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(ignore_https_errors=True)
        page = context.new_page()
        contents = _handle_children_url(
            page,
            url,
            interval_time,
            resource_types,
            max_depth,
            max_count,
            exclude_sub_urls,
            include_sub_urls,
            exclude_img_info
        )
        context.close()
        browser.close()
        logger.debug(f"{log_tag_const.WEB_CRAWLING} end time {date_time_utils.now_str()}")
        return contents


def _handle_children_url(
        browser_page,
        url,
        interval_time,
        resource_types,
        max_depth,
        max_count,
        exclude_sub_urls,
        include_sub_urls,
        exclude_img_info
):
    """Recursively crawling the children urls, images urls on a website

    Args:
        page(Page):
        url (str): The URL to crawl.
        interval_time (int): The interval time between crawling web pages.
        resource_types (list): Crawling the types of resource on a web page.
        max_depth (int): The max depth of the recursive loading.
        max_count (int): The max web count of the recursive loading.
        exclude_sub_urls (list): Excluding subpage urls while crawling a website.
        include_sub_urls (list): Including subpage urls while crawling a website.
        exclude_img_info (dict): Excluding images with specified attributes (eg: weight、height) while crawling a website.
    Returns:
         [
            {
                "id": "9D4oAQWh",
                "pid": "0",
                "url": "https://mp.weixin.qq.com/s/65DFY...",
                "type": "page",
                "level": 1
            }
        ]
    Examples:
        {
            "brower_page": 
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
            "interval_time": 1,
            "resource_types": [
                "png",
                "jpg"
            ],
            "max_depth": 1,
            "max_count": 100,
            "exclude_sub_urls": [
                ""
            ],
            "include_sub_urls": [
                ""
            ]
        }
    Raises:
    """
    logger.debug(f"{log_tag_const.WEB_CRAWLING} Loading Root URL: {url}")
    contents = []
    count = 1
    # 已爬取网页URL
    visited = set()
    children_urls = [
        {
            "url": url,
            "pid": "root"
        }
    ]
    children_urls_bak = []
    for i in range(0, max_depth):
        if len(children_urls) == 0:
            logger.debug(f'{log_tag_const.WEB_CRAWLING} Crawling completed, no subpages found')
            return contents
        for children_url in children_urls:
            id = ulid.ulid()
            if count > max_count:
                logger.debug(
                    f'{log_tag_const.WEB_CRAWLING} Crawling completed, exceeding the maximum limit for the number({max_count}) of web pages to be crawled'
                )
                break
            content = _handle_content(
                    browser_page,
                    children_url["url"],
                    resource_types,
                    exclude_sub_urls,
                    include_sub_urls,
                    exclude_img_info,
                    visited,
                    id,
                    i+1
            )
            if bool(content):
                count = count + 1
            else:
                continue
            logger.debug(f"{log_tag_const.WEB_CRAWLING} Crawling completed, content {content}" )
            # 添加层级属性
            content["id"] = id
            content["pid"] = children_url["pid"]
            content["level"] = i + 1

            if content["children_urls"] is not None:
                for url in content['children_urls']:
                    children_urls_bak.append({
                        "pid": id, 
                        "url": url
                    }
                )
            visited.add(children_url["url"])

            contents.extend(content["images"])
            content.pop("images")
            content.pop("children_urls")
            contents.append(content)
            # 时间间隔
            time.sleep(interval_time)
        children_urls = []
        children_urls = children_urls_bak
        children_urls_bak = []
    return contents

def _handle_content(
    browser_page,
    url,
    resource_types,
    exclude_sub_urls,
    include_sub_urls,
    exclude_img_info,
    visited,
    id,
    level
):
    """Crawling the children urls, image urls on a URL

    Args:
        browser_page(Page): Playwright page
        url (str): The URL to crawl.
        interval_time (int): The interval time between crawling web pages.
        resource_types (list): Crawling the types of resource on a web page.
        exclude_sub_urls (list): Excluding subpage urls while crawling a website.
        include_sub_urls (list): Including subpage urls while crawling a website.
        visited(list): A collection of crawled web pages
        id(str): The current page ID of the image
        level(int): The current page level of the image
        exclude_img_info (dict): Excluding images with specified attributes (eg: weight、height) while crawling a website.
    Returns:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ"
            "images": [
                 {
                     "url": "https://mmbiz.qpic.cn/sz_mmbiz_png/KmXPKA19gW8nMO7tY8sbhvVXDj9SjZPCibOZMwwiauibXxTlwGr3ic1PEjEiaU69Xa2RMUuWnCvz99ZyDgb7OzEAR9w/640?wx_fmt=png&from=appmsg&wxfrom=5&wx_lazy=1&wx_co=1",
                    "id": "9D4oAQWh",
                    "pid": "0",
                    "type": "image",
                    "level": 1
                },
            ]
            "children_urls": [
                "url": "https://mp.weixin.qq.com/s/9D4oAQ...."
            ]
        }
    Examples:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
            "interval_time": 1,
            "resource_types": [
                "png",
                "jpg"
            ],
            "exclude_sub_urls": [
                ""
            ],
            "include_sub_urls": [
                ""
            ],
            
        }
    Raises:
    """
    content = {}
    content['url'] = url
    content["type"] = "page"
    children_urls = []
    images = []
    try:
        for include_sub_url in include_sub_urls:
            if not fnmatch.fnmatch(urlparse(url).path, include_sub_url):
                logger.debug(f"{log_tag_const.WEB_CRAWLING} Not Include Loading Children URL : {url}")
                return {}
        for exclude_sub_url in exclude_sub_urls:
            if fnmatch.fnmatch(urlparse(url).path, exclude_sub_url):
                logger.debug(f"{log_tag_const.WEB_CRAWLING} Exclude Loading Children URL : {url}")
                return {}
        logger.debug(f"{log_tag_const.WEB_CRAWLING} Loading Children URL : {url}")
        browser_page.goto(url)
        #处理错误响应码
        def when_response(response):
            if response.status >= 400 and response.request.url == url:
                logger.error(
                    "".join(
                        [
                            f"{log_tag_const.WEB_CRAWLING} Loading url {url} failure, ",
                            f"status {response.status}, "
                            f"error message {response.text}"
                        ]
                    )
                )
                content['status'] = response.status
                content['error_message'] = response.text
        browser_page.on('response', when_response)
        # 图片
        images = _handle_resource(
            browser_page,
            url,
            resource_types,
            exclude_img_info,
            id,
            level
        )
        visited.add(url)
        all_links = [urljoin(url, link.get_attribute("href")) for link in browser_page.query_selector_all('//a')]
        #child_links = [link for link in set(all_links) if link.startswith(url)]
        child_links = [link for link in set(all_links) if link.startswith("http")]
        # Remove fragments to avoid repetitions
        defraged_child_links = [urldefrag(link).url for link in set(child_links)]
        for link in set(defraged_child_links):
            if link not in visited:
                visited.add(link)
                children_urls.append(link)
        logger.debug(f"Children urls: {children_urls}")
    except Exception:
        content["status"] = 400
        content["error_message"] = traceback.format_exc()
        logger.error(
            ''.join(
                [
                    f"{log_tag_const.WEB_CRAWLING} Execute crawling url failure\n",
                    f"The tracing error is: \n{traceback.format_exc()}"
                ]
            )
        )
    content['children_urls'] = children_urls
    content["images"] = images
    return content

def handle_content(
    url
):
    """Crawling the content, and other resources on a URL
    Args:
        url (str): The URL to crawl.  
    Returns:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ"
            "title": "专为数据库打造：DB-GPT用私有化LLM技术定义数据库下一代交互方式",
            "text": "2023 年 6 ...",
            "html": "<html> ...",
            "description": "专为数据库打造：DB-GPT用私 ...",
            "language": "en"
        }
    Examples:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
        }
    Raises:
    """
    logger.debug(f"{log_tag_const.WEB_CRAWLING} Loading Root URL Detail: {url}")
    content = {}
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch()
            context = browser.new_context(ignore_https_errors=True)
            page = context.new_page()
            page.goto(url)
            innerHTML = page.content()
            soup = BeautifulSoup(innerHTML, "html.parser")
            # URL
            content['source'] = url
            # 标题
            title = soup.find('title').get_text()
            content['title'] = title
            # 源网页
            content["html"] = innerHTML
            # 内容
            text = soup.get_text()
            content['text'] = text
            # 描述
            description = soup.find("meta", attrs={"name": "description"})
            if description is not None:
                content["description"] = description.get("content", "No description found.")
            # 语言
            language = soup.find("html").get("lang", "No language found.")
            content["language"] = language
            logger.debug(f"{log_tag_const.WEB_CRAWLING} Loading content: {content}")

            context.close()
            browser.close()
            return content
    except Exception:
        logger.error(''.join([
            f"{log_tag_const.WEB_CRAWLING} Execute crawling url failure\n",
            f"The tracing error is: \n{traceback.format_exc()}"
        ]))
    return content

def _handle_resource(
        browser_page,
        url,
        resource_types,
        exclude_img_info,
        pid,
        level
):
    """Crawling images urls on a URL
    Args:
        brower_page(Page): Playwright page
        url (str): The URL to crawl.
        resource_types (list): Crawling the types of resource on a web page.
        exclude_img_info (dict): Excluding images with specified attributes (eg: weight、height) while crawling a website.
        pid(str): The parent page ID of the image
        level(int): The parent page level of the image
    Returns:
        [
            {
                "id": "",
                "pid": "0",
                "url": "https://mp.weixin.qq.com/s/9D4oAQW ...",
                "type": "image",
                "level": "1"
            }
        ]
    Examples:
        {
            "page": Playwright Page 
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
            "resource_types": [
                "png",
                "jpg"
            ],
            "pid": "0",
            "level": "1"
        }  
    Raises:
    """
    resources = []
    if len(resource_types) == 0:
        return resources
    pic_src = browser_page.query_selector_all('//img')
    for pic in pic_src:
        image = {}
        # 处理懒加载问题
        pic_url = pic.get_attribute('data-src')
        if pic_url is None or pic_url == '':
            pic_url = pic.get_attribute('src')
        logger.debug(f"{log_tag_const.WEB_CRAWLING} Parse Image Url: {pic_url}")
        if pic_url is None or pic_url == '':
            continue
        img_all_url = pic_url if pic_url.startswith('http') else urljoin(url, pic_url)
        if not filter_image(img_all_url, resource_types, exclude_img_info):
            continue
        image["url"] = img_all_url
        image["id"] = ulid.ulid()
        image["pid"] = pid
        image["type"] = "image"
        image["level"] = level
        resources.append(image)
    return resources


def filter_image(url, resource_types, exclude_img_info):
    """ Filter out ineligible images
    Args:
        url (str): The Image URL.
        resource_types (list): Crawling the types of resource on a web page.
        exclude_img_info (dict): Excluding images with specified attributes (eg: weight、height) while crawling a website.
    Returns:
    Examples:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
            "resource_types": [
                "png",
                "jpg"
            ],
            "exclude_img_info": {
                "weight": 250,
                "height": 250
            }
            
        }
    Raises:
    """
    try:
         # 下载图片数据
        response = requests.get(url)
        # 检查响应状态码是否为200，表示请求成功
        if response.status_code == 200:
            # 将图片数据转换为BytesIO对象以便于PIL处理
            img_data = BytesIO(response.content)
            # 使用PIL打开并识别图片类型
            try:
                image = Image.open(img_data)
                weight, height = image.size
                if weight < exclude_img_info["weight"] and height < exclude_img_info["height"]:
                    logger.debug(
                        "".join(
                            [
                                f"{log_tag_const.WEB_CRAWLING} Smaller than the size limit for crawled images\n",
                                f" Original weight: {weight}, Original height: {height}"
                            ]
                        )
                    )
                    return False
                # 如果format无法获取，则默认为JPEG格式
                format = image.format or 'JPEG'
                if  format.lower() not in resource_types:
                    logger.debug(f"{log_tag_const.WEB_CRAWLING} Not within the range of resource types to be crawled")
                    return False
                return True
            except IOError:
                logger.error(
                    ''.join(
                        [
                            f"{log_tag_const.WEB_CRAWLING} Unable to recognize or open the image\n",
                            f"The tracing error is: \n{traceback.format_exc()}"
                        ]
                    )
                )
        else:
            logger.error(f"Request failed, HTTP status code：{response.status_code}")
            return False
    except Exception:
        logger.error(''.join([
             f"{log_tag_const.WEB_CRAWLING} Execute Request Image Failure\n",
             f"The tracing error is: \n{traceback.format_exc()}"
         ]))
        return False

def download_and_save_image(url, image_id):
    """ Save Image to Local
    Args:
        url (str): Image URL.
        image_id (str): Image ID.
    Returns:
    Examples:
        {
            "url": "https://mp.weixin.qq.com/s/9D4oAQWh-JjWis2MYEY_gQ",
            "img_id": "9D4oAQW..."
        }  
    Raises:
    """
    logger.debug(f"Download Image URL : {url}")
    try:
        # 下载图片数据
        response = requests.get(url)
        # 检查响应状态码是否为200，表示请求成功
        if response.status_code == 200:
            # 将图片数据转换为BytesIO对象以便于PIL处理
            img_data = BytesIO(response.content)
            # 使用PIL打开并识别图片类型
            try:
                image = Image.open(img_data)
                format = image.format or 'JPEG'  # 如果format无法获取，则默认为JPEG格式
            except IOError:
                logger.error(
                    ''.join(
                        [
                            f"{log_tag_const.WEB_CRAWLING} Unable to recognize or open the image\n",
                            f"The tracing error is: \n{traceback.format_exc()}"
                        ]
                    )
                )
                return None
            output_filename = f"{image_id}.{format.lower()}"
            # 保存图片到本地
            try:
                image.save(output_filename)
                logger.debug(f"{log_tag_const.WEB_CRAWLING} The image has been successfully downloaded and saved as：{output_filename}")
                return output_filename
            except Exception:
                logger.error(
                    ''.join(
                        [
                            f"{log_tag_const.WEB_CRAWLING} Execute save the image failure\n",
                            f"The tracing error is: \n{traceback.format_exc()}"
                        ]
                    )
                )
                return None
        else:
            logger.error(f"{log_tag_const.WEB_CRAWLING} Request failed, HTTP status code：{response.status_code}")
            return None
    except Exception:
        logger.error(
            ''.join(
                [
                    f"{log_tag_const.WEB_CRAWLING} Execute Request Image Failure\n",
                    f"The tracing error is: \n{traceback.format_exc()}"
                ]
            )
        )
        return None
