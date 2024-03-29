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

source ./scripts/utils.sh ./scripts/setup_testenv.sh
source ./scripts/verify_datasource.sh ./scripts/verify_vectorstore.sh \
	./scripts/verify_embedder.sh  ./scripts/verify_knowledgebase.sh \
	./scripts/verify_app.sh

if [[ $RUNNER_DEBUG -eq 1 ]] || [[ $GITHUB_RUN_ATTEMPT -gt 1 ]]; then
	# use [debug logging](https://docs.github.com/en/actions/monitoring-and-troubleshooting-workflows/enabling-debug-logging)
	# or run the same test multiple times.
	set -x
fi
export TERM=xterm-color

TempFilePath=${TempFilePath:-"/tmp/kubeagi-example-test"}
InstallDirPath=${TempFilePath}/building-base
DefaultPassWord=${DefaultPassWord:-'passw0rd'}
LOG_DIR=${LOG_DIR:-"/tmp/kubeagi-example-test/logs"}
RootPath=$(dirname -- "$(readlink -f -- "$0")")/..
portal_pid=0
RETRY_COUNT=5

mkdir ${TempFilePath} || true
env

trap 'debugInfo $LINENO' ERR
trap 'debugInfo $LINENO' EXIT
debug=0

info "1. setup test env"
setup_testenv

info "2. load reranking image if not exist"
rerank_image="kubeagi/core-library-cli:v0.0.1-20240308-18ea8aa"
docker pull $rerank_image
kind load docker-image $rerank_image --name=$KindName
if [[ $GITHUB_ACTIONS == "true" ]]; then
	# github action has less disk space
	docker rmi $rerank_image
fi
df -h

info "3. verify datasource"
verify_datasource

info "4. verify vectorstore"
verify_vectorstore

info "5. verify versioned dataset"
verify_dataset

info "6. verify a 3rd party embedder"
verify_3rd_party_embedder

info "7.verify knowledgebase"
verify_knowledgebase

info "8. verify knowledgebase force reconcile"
verify_knowledgebase_force_reconcile


info "8 validate simple app can work normally"
verify_app

info "9. show apiserver logs for debug"
kubectl logs --tail=100 -n arcadia -l app=arcadia-apiserver >/tmp/apiserver.log
cat /tmp/apiserver.log

info "all finished! ✅"
