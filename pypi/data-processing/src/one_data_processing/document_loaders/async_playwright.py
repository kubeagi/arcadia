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
import traceback
from typing import List

from langchain_community.document_transformers import Html2TextTransformer
from langchain_core.documents import Document
from playwright.async_api import async_playwright

from common import log_tag_const
from document_loaders.base import BaseLoader

logger = logging.getLogger(__name__)

class AsyncPlaywrightLoader(BaseLoader):
    """Scrape HTML pages from URLs using a
    headless instance of the Chromium."""

    def __init__(
        self,
        url: str,
        max_count: int = 100,
        max_depth: int = 1,
        interval_time: int = 1000,
    ):
        """
        Initialize the loader with a list of URL paths.

        Args:
            url (str): Website url.
            max_count (int): Maximum Number of Website URLs.
            max_depth (int): Website Crawling Depth.
            interval_time (int): Interval Time.
        """
        if max_count is None:
            max_count = 100
        if max_depth is None:
            max_depth = 1
        if interval_time is None:
            interval_time = 1000

        self._url = url
        self._max_count = max_count
        self._max_depth = max_depth
        self._interval_time = interval_time / 1000

    async def ascrape_playwright(self, url: str) -> str:
        """
        Asynchronously scrape the content of a given URL using Playwright's async API.

        Args:
            url (str): The URL to scrape.

        Returns:
            str: The scraped HTML content or an error message if an exception occurs.

        """
        logger.info("Starting scraping...")

        results = ""
        async with async_playwright() as p:
            browser = await p.chromium.launch(headless=True)
            try:
                page = await browser.new_page()
                await page.goto(url)
                results = await page.content()  # Simply get the HTML content
                logger.info("Content scraped")
            except Exception as e:
                results = f"Error: {e}"
            await browser.close()
        return results

    async def load(self) -> List[Document]:
        """
        Load and return all Documents from the provided URLs.

        Returns:
            List[Document]: A list of Document objects
            containing the scraped content from each URL.

        """
        logger.info(f"{log_tag_const.WEB_LOADER} Async start to load Website data")

        docs = []
        all_url = await self.get_all_url()
        for url in all_url:
            html_content = await self.ascrape_playwright(url)
            metadata = {"source": url, "page": 0}
            docs.append(Document(page_content=html_content, metadata=metadata))

        html2text = Html2TextTransformer()
        docs_transformed = html2text.transform_documents(docs)
        return docs_transformed

    async def get_all_url(self):
        """
        Retrieve the URLs for Data Extraction from the Website.

        Args:
            url (str): Website url.
            max_count (int): Maximum Number of Website URLs.
            max_depth (int): Website Crawling Depth.
            interval_time (int): Interval Time.

        """
        logger.debug(
            "".join(
                [
                    f"{log_tag_const.WEB_CRAWLING} Get all url in a web page\n",
                    f"  url: {self._url}"
                ]
            )
        )

        all_url = [self._url]
        sub_urls = [self._url]
        try:
            for _ in range(1, self._max_depth):
                for sub_url in sub_urls:
                    children_urls = await self._get_children_url(
                        url=sub_url,
                        url_count=len(all_url)
                    )

                    if children_urls.get("status") == 200:
                        res = children_urls.get("data")

                        # 避免重复的url
                        unique_urls = set(all_url)
                        unique_urls.update(res.get("children_url"))
                        all_url = list(unique_urls)

                        # 如果达到最大数量限制，直接返回
                        if res.get("url_count") >= self._max_count:
                            logger.info(
                                "".join(
                                    [
                                        f"{log_tag_const.WEB_CRAWLING} The number of URLs has reached the upper limit.\n",
                                        f"  max_count: {self._max_count}\n"
                                    ]
                                )
                            )
                            return all_url

                        sub_urls = res.get("children_url")
                        # 时间间隔
                        logger.info(f"{log_tag_const.WEB_CRAWLING} Wait for {self._interval_time} seconds before continuing the visit.")
                        time.sleep(self._interval_time)
            return all_url
        except Exception:
            logger.error(
                ''.join(
                    [
                        f"{log_tag_const.WEB_CRAWLING} Execute crawling url failure\n",
                        f"The tracing error is: \n{traceback.format_exc()}"
                    ]
                )
            )
            return all_url

    async def _get_children_url(self, url, url_count):
        """
        Retrieve URLs contained in the website.

        Args:
            url (str): Website url.
            url_count (int): URL count.

        """
        logger.debug(
            "".join(
                [
                    f"{log_tag_const.WEB_CRAWLING} Get sub url in a web page\n",
                    f"  url: {url}\n",
                    f"  max_count: {self._max_count}\n",
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
                    if url_count >= self._max_count:
                        logger.info(
                            "".join(
                                [
                                    f"{log_tag_const.WEB_CRAWLING} The number of URLs has reached the upper limit.\n",
                                    f"  max_count: {self._max_count}\n",
                                    f"  url_count: {url_count}"
                                ]
                            )
                        )
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
            return {"status": 500, "message": "获取网页中的子网页url失败", "data": ""}
