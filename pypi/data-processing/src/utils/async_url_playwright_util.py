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
import time
import asyncio
import traceback
from playwright.async_api import async_playwright

from common import log_tag_const

logger = logging.getLogger(__name__)

async def get_all_url(url, max_count, max_depth, interval_time):
    logger.debug(
        "".join(
            [
                f"{log_tag_const.WEB_CRAWLING} Get all url in a web page\n",
                f"  url: {url}"
            ]
        )
    )

    try:
        all_url = [url]
        sub_urls = [url]
        for i in range(1, max_depth):
            for url in sub_urls:
                children_urls = await _get_children_url(
                    url=url,
                    max_count=max_count,
                    url_count=len(all_url)
                )

                if children_urls.get("status") == 200:
                    res = children_urls.get("data")

                    # 避免重复的url
                    unique_urls = set(all_url)
                    unique_urls.update(res.get("children_url"))
                    all_url = list(unique_urls)
                    # all_url.extend(res.get("children_url"))

                    if res.get("url_count") >= max_count:
                        return {"status": 200, "message": "", "data": all_url}
                    
                    sub_urls = res.get("children_url")
                    # 时间间隔
                    time.sleep(interval_time)
        return {"status": 200, "message": "", "data": all_url}
    except Exception:
        logger.error(
            ''.join(
                [
                    f"{log_tag_const.WEB_CRAWLING} Execute crawling url failure\n",
                    f"The tracing error is: \n{traceback.format_exc()}"
                ]
            )
        )
        return {"status": 400, "message": "获取网页中的子网页url失败", "data": ""}


async def _get_children_url(url, max_count, url_count):
    logger.debug(
        "".join(
            [
                f"{log_tag_const.WEB_CRAWLING} Get sub url in a web page\n",
                f"  url: {url}\n",
                f"  max_count: {max_count}\n",
                f"  url_count: {url_count}"
            ]
        )
    )

    try:
        children_url = []
        async with async_playwright() as p:
            browser = await p.chromium.launch()
            context = await browser.new_context()
            page = await context.new_page()

            # 在浏览器中打开网页
            await page.goto(url)

            # 提取每个 a 标签的 href 属性
            links = await page.query_selector_all('a')
            for link in links:
                href = await link.get_attribute('href')
                # 需要抓取的url数量不得超过最大数量
                if url_count >= max_count:
                    break

                # 获取以http开头的url 并排除已存在的url
                if href:
                    if href.startswith("http") and href not in children_url:
                        children_url.append(href)
                        url_count += 1

            # 关闭浏览器
            await browser.close()
        data = {
            "children_url": children_url,
            "url_count": url_count
        }
        return {"status": 200, "message": "", "data": data}
    except Exception:
        logger.error(
            ''.join(
                [
                    f"{log_tag_const.WEB_CRAWLING} Execute crawling url failure\n",
                    f"The tracing error is: \n{traceback.format_exc()}"
                ]
            )
        )
        return {"status": 400, "message": "获取网页中的子网页url失败", "data": ""}



