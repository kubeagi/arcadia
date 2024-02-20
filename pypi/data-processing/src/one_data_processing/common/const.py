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

LLM_WAIT_SECONDS = 120

CLEAN_SUPPORT_TYPE = [
    "remove_invisible_characters",
    "space_standardization",
    "remove_garbled_text",
    "traditional_to_simplified",
    "remove_html_tag",
    "remove_emojis",
]
PRIVACY_SUPPORT_TYPE = ["remove_email", "remove_ip_address", "remove_number"]
