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
if [[ $RUNNER_DEBUG -eq 1 ]] || [[ $GITHUB_RUN_ATTEMPT -gt 1 ]]; then
	# use [debug logging](https://docs.github.com/en/actions/monitoring-and-troubleshooting-workflows/enabling-debug-logging)
	# or run the same test multiple times.
	set -x
fi
export TERM=xterm-color

KindName="kubeagi"
TimeoutSeconds=${TimeoutSeconds:-"600"}
HelmTimeout=${HelmTimeout:-"1800s"}
KindVersion=${KindVersion:-"v1.24.4"}
TempFilePath=${TempFilePath:-"/tmp/kubeagi-example-test"}
KindConfigPath=${TempFilePath}/kind-config.yaml
InstallDirPath=${TempFilePath}/building-base
DefaultPassWord=${DefaultPassWord:-'passw0rd'}
LOG_DIR=${LOG_DIR:-"/tmp/kubeagi-example-test/logs"}
RootPath=$(dirname -- "$(readlink -f -- "$0")")/..

Timeout="${TimeoutSeconds}s"
mkdir ${TempFilePath} || true

function debugInfo {
	if [[ $? -eq 0 ]]; then
		exit 0
	fi
	if [[ $debug -ne 0 ]]; then
		exit 1
	fi

	warning "debugInfo start 🧐"
	mkdir -p $LOG_DIR

	warning "1. Try to get all resources "
	kubectl api-resources --verbs=list -o name | xargs -n 1 kubectl get -A --ignore-not-found=true --show-kind=true >$LOG_DIR/get-all-resources-list.log
	kubectl api-resources --verbs=list -o name | xargs -n 1 kubectl get -A -oyaml --ignore-not-found=true --show-kind=true >$LOG_DIR/get-all-resources-yaml.log

	warning "2. Try to describe all resources "
	kubectl api-resources --verbs=list -o name | xargs -n 1 kubectl describe -A >$LOG_DIR/describe-all-resources.log

	warning "3. Try to export kind logs to $LOG_DIR..."
	kind export logs --name=${KindName} $LOG_DIR
	sudo chown -R $USER:$USER $LOG_DIR

	warning "debugInfo finished ! "
	warning "This means that some tests have failed. Please check the log. 🌚"
	debug=1
	exit 1
}
trap 'debugInfo $LINENO' ERR
trap 'debugInfo $LINENO' EXIT
debug=0

function cecho() {
	declare -A colors
	colors=(
		['black']='\e[0;47m'
		['red']='\e[0;31m'
		['green']='\E[0;32m'
		['yellow']='\E[0;33m'
		['blue']='\E[0;34m'
		['magenta']='\E[0;35m'
		['cyan']='\E[0;36m'
		['white']='\E[0;37m'
	)
	local defaultMSG="No message passed."
	local defaultColor="black"
	local defaultNewLine=true
	while [[ $# -gt 1 ]]; do
		key="$1"
		case $key in
		-c | --color)
			color="$2"
			shift
			;;
		-n | --noline)
			newLine=false
			;;
		*)
			# unknown option
			;;
		esac
		shift
	done
	message=${1:-$defaultMSG}     # Defaults to default message.
	color=${color:-$defaultColor} # Defaults to default color, if not specified.
	newLine=${newLine:-$defaultNewLine}
	echo -en "${colors[$color]}"
	echo -en "$message"
	if [ "$newLine" = true ]; then
		echo
	fi
	tput sgr0 #  Reset text attributes to normal without clearing screen.
	return
}

function warning() {
	cecho -c 'yellow' "$@"
}

function error() {
	cecho -c 'red' "$@"
}

function info() {
	cecho -c 'blue' "$@"
}

function waitPodReady() {
	namespace=$1
	podLabel=$2
	START_TIME=$(date +%s)
	while true; do
		readStatus=$(kubectl -n${namespace} get po -l ${podLabel} --ignore-not-found=true -o json | jq -r '.items[0].status.conditions[] | select(."type"=="Ready") | .status')
		if [[ $readStatus == "True" ]]; then
			info "Pod:${podLabel} ready"
			kubectl -n${namespace} get po -l ${podLabel}
			break
		fi

		CURRENT_TIME=$(date +%s)
		ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
		if [ $ELAPSED_TIME -gt $TimeoutSeconds ]; then
			error "Timeout reached"
			kubectl describe po -n${namespace} -l ${podLabel}
			kubectl get po -n${namespace} --show-labels
			exit 1
		fi
		sleep 5
	done
}

function waitCRDStatusReady() {
	source=$1
	namespace=$2
	name=$3
	START_TIME=$(date +%s)
	while true; do
		readStatus=$(kubectl -n${namespace} get ${source} ${name} --ignore-not-found=true -o json | jq -r '.status.conditions[0].status')
		message=$(kubectl -n${namespace} get ${source} ${name} --ignore-not-found=true -o json | jq -r '.status.conditions[0].message')
		if [[ $readStatus == "True" ]]; then
			info $message
			break
		fi

		CURRENT_TIME=$(date +%s)
		ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
		if [ $ELAPSED_TIME -gt $TimeoutSeconds ]; then
			error "Timeout reached"
			exit 1
		fi
		sleep 5
	done
}

info "1. create kind cluster"
read -r -p "Is kind cluster created? [Y/n]" input
    case $input in 
	    [yY][eE][sS]|[yY]|"")
		echo kind cluster created, skip this setp.
		;;
		[nN][oO]|[nN])
        make kind
		;;
    *)
        echo "Invalid input..."
		exit 1
        ;;
esac

info "2. load arcadia image to kind"
docker tag controller:latest controller:example-e2e
kind load docker-image controller:example-e2e --name=$KindName

info "3. install arcadia"
kubectl create namespace arcadia
helm install -narcadia arcadia charts/arcadia --set deployment.image=controller:example-e2e --wait --timeout $HelmTimeout

info "4. check system datasource arcadia-minio(system datasource)"
waitCRDStatusReady "Datasource" "arcadia" "arcadia-minio"

info "5. create and verify local datasource"
kubectl apply -f config/samples/arcadia_v1alpha1_local_datasource.yaml
waitCRDStatusReady "Datasource" "arcadia" "arcadia-local"
datasourceType=$(kubectl get datasource -n arcadia arcadia-local -o=jsonpath='{.metadata.labels.arcadia\.kubeagi\.k8s\.com\.cn/datasource-type}')
if [[ $datasourceType != "local" ]]; then
	error "Datasource should local but got $datasourceType"
	exit 1
fi

info "6. create and verify vectorstore"
info "6.1. helm install chroma"
helm repo add chroma https://amikos-tech.github.io/chromadb-chart/
helm repo update chroma
helm install -narcadia chroma chroma/chromadb --set service.type=ClusterIP --set chromadb.auth.enabled=false --wait --timeout $HelmTimeout
info "6.2. verify chroma vectorstore status"
kubectl apply -f config/samples/arcadia_v1alpha1_vectorstore.yaml
waitCRDStatusReady "VectorStore" "arcadia" "chroma-sample"

info "7. create and verify knowledgebase"
info "7.1. upload some test file to system datasource"
bucket=$(kubectl get datasource -n arcadia arcadia-minio -o json | jq -r .spec.oss.bucket)
s3_key=$(kubectl get secrets -n arcadia arcadia-minio -o json | jq -r ".data.rootUser" | base64 --decode)
s3_secret=$(kubectl get secrets -n arcadia arcadia-minio -o json | jq -r ".data.rootPassword" | base64 --decode)
resource="/${bucket}/example-test/knowledgebase-1.txt"
content_type="application/octet-stream"
date=$(date -R)
_signature="PUT\n\n${content_type}\n${date}\n${resource}"
signature=$(echo -en ${_signature} | openssl sha1 -hmac ${s3_secret} -binary | base64)
kubectl port-forward -n arcadia svc/arcadia-minio 9000:9000 >/dev/null 2>&1 &
minio_pid=$!
info "port-forward minio in pid: $minio_pid"
sleep 3
curl -X PUT -T "tests/knowledgebase-1.txt" \
	-H "Host: 127.0.0.1:9000" \
	-H "Date: ${date}" \
	-H "Content-Type: ${content_type}" \
	-H "Authorization: AWS ${s3_key}:${signature}" \
	http://127.0.0.1:9000${resource}
info "7.2. create embedder and wait it ready"
kubectl apply -f config/samples/arcadia_v1alpha1_embedders.yaml
waitCRDStatusReady "Embedders" "arcadia" "zhipuai-embedders-sample"
info "7.3. create knowledgebase and wait it ready"
kubectl apply -f config/samples/arcadia_v1alpha1_knowledgebase.yaml
waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample"
info "7.4. check this vectorstore has data"
kubectl port-forward -n arcadia svc/chroma-chromadb 8000:8000 >/dev/null 2>&1 &
chroma_pid=$!
info "port-forward chroma in pid: $minio_pid"
sleep 3
collection_test_id=$(curl http://127.0.0.1:8000/api/v1/collections/arcadia_knowledgebase-sample | jq -r .id)
collection_test_count=$(curl http://127.0.0.1:8000/api/v1/collections/${collection_test_id}/count)
if [[ $collection_test_count =~ ^[0-9]+$ ]]; then
	info "collection test count: $collection_test_count"
else
	echo "$collection_test_count is not a number"
	exit 1
fi

info "all finished! ✅"
