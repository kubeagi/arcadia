#!/bin/bash
#
# Copyright contributors to the KubeAGI project
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
# 	  http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

source ./scripts/utils.sh

# implement a shell function to list all images under docker repo https://hub.docker.com/r/kubeagi/
# https://docs.docker.com/docker-hub/api/latest/#tag/repositories/paths/~1v2~1namespaces~1%7Bnamespace%7D~1repositories~1%7Brepository%7D~1tags/get
list_docker_images() {
    local api_url="https://hub.docker.com/v2/namespaces/kubeagi/repositories/${repo}/"

    # Fetch the repository details using the Docker Hub API
    local response=$(curl -s "${api_url}")

    # Extract the image names and tags from the response using jq (make sure jq is installed)
    local image_tags=$(echo "${response}" | jq -r '.results[] | .name + ": " + .tags[0].name')

    # Print the list of image names and latest tags
    info "Images under ${repo}:"
    info "${image_tags}"
}

list_docker_images