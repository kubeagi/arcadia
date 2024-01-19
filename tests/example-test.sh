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
TimeoutSeconds=${TimeoutSeconds:-"300"}
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
	if [[ $GITHUB_ACTIONS == "true" ]]; then
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
	fi
	exit 1
}
trap 'debugInfo $LINENO' ERR
trap 'debugInfo $LINENO' EXIT
debug=0

function cecho() {
	declare -A colors
	colors=(
		['black']='\E[0;47m'
		['red']='\E[0;31m'
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
			if [[ ${source} == "KnowledgeBase" ]]; then
				fileStatus=$(kubectl get knowledgebase -n $namespace $name -o json | jq -r '.status.fileGroupDetail[0].fileDetails[0].phase')
				if [[ $fileStatus != "Succeeded" ]]; then
					kubectl get knowledgebase -n $namespace $name -o json | jq -r '.status.fileGroupDetail[0].fileDetails'
					exit 1
				fi
			fi
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

function getRespInAppChat() {
	appname=$1
	namespace=$2
	query=$3
	conversationID=$4
	testStream=$5
	START_TIME=$(date +%s)
	while true; do
		data=$(jq -n --arg appname "$appname" --arg query "$query" --arg namespace "$namespace" --arg conversationID "$conversationID" '{"query":$query,"response_mode":"blocking","conversation_id":$conversationID,"app_name":$appname, "app_namespace":$namespace}')
		resp=$(curl -s -XPOST http://127.0.0.1:8081/chat --data "$data")
		ai_data=$(echo $resp | jq -r '.message')
		references=$(echo $resp | jq -r '.references')
		if [ -z "$ai_data" ] || [ "$ai_data" = "null" ]; then
			echo $resp
			exit 1
		fi
		echo "👤: ${query}"
		echo "🤖: ${ai_data}"
		echo "🔗: ${references}"
		resp_conversation_id=$(echo $resp | jq -r '.conversation_id')

		if [ $testStream == "true" ]; then
			info "just test stream mode"
			data=$(jq -n --arg appname "$appname" --arg query "$query" --arg namespace "$namespace" --arg conversationID "$conversationID" '{"query":$query,"response_mode":"streaming","conversation_id":$conversationID,"app_name":$appname, "app_namespace":$namespace}')
			curl -s -XPOST http://127.0.0.1:8081/chat --data "$data"
		fi
		break
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
make kind

info "2. load arcadia image to kind"
docker tag controller:latest controller:example-e2e
kind load docker-image controller:example-e2e --name=$KindName

info "3. install arcadia"
kubectl create namespace arcadia
helm install -narcadia arcadia deploy/charts/arcadia -f tests/deploy-values.yaml \
	--set controller.image=controller:example-e2e --set apiserver.image=controller:example-e2e \
	--wait --timeout $HelmTimeout

info "4. check system datasource arcadia-minio(system datasource)"
waitCRDStatusReady "Datasource" "arcadia" "arcadia-minio"

info "5. create and verify datasource"
info "5.1 oss datasource"
kubectl apply -f config/samples/arcadia_v1alpha1_datasource.yaml
waitCRDStatusReady "Datasource" "arcadia" "datasource-sample"
datasourceType=$(kubectl get datasource -n arcadia datasource-sample -o=jsonpath='{.metadata.labels.arcadia\.kubeagi\.k8s\.com\.cn/datasource-type}')
if [[ $datasourceType != "oss" ]]; then
	error "Datasource should be oss but got $datasourceType"
	exit 1
fi
info "5.2 PostgreSQL datasource"
kubectl apply -f config/samples/arcadia_v1alpha1_datasource_postgresql.yaml
waitCRDStatusReady "Datasource" "arcadia" "datasource-postgresql-sample"
datasourceType=$(kubectl get datasource -n arcadia datasource-postgresql-sample -o=jsonpath='{.metadata.labels.arcadia\.kubeagi\.k8s\.com\.cn/datasource-type}')
if [[ $datasourceType != "postgresql" ]]; then
	error "Datasource should be oss but got $datasourceType"
	exit 1
fi

info "6. verify default vectorstore"
waitCRDStatusReady "VectorStore" "arcadia" "arcadia-pgvector-vectorstore"
info "6.2 verify PGVector vectorstore"
kubectl apply -f config/samples/arcadia_v1alpha1_vectorstore_pgvector.yaml
waitCRDStatusReady "VectorStore" "arcadia" "pgvector-sample"

info "7. create and verify knowledgebase"

info "7.1. upload some test file to system datasource"
kubectl port-forward -n arcadia svc/arcadia-minio 9000:9000 >/dev/null 2>&1 &
minio_pid=$!
sleep 3
info "port-forward minio in pid: $minio_pid"
bucket=$(kubectl get datasource -n arcadia datasource-sample -o json | jq -r .spec.oss.bucket)
s3_key=$(kubectl get secrets -n arcadia datasource-sample-authsecret -o json | jq -r ".data.rootUser" | base64 --decode)
s3_secret=$(kubectl get secrets -n arcadia datasource-sample-authsecret -o json | jq -r ".data.rootPassword" | base64 --decode)
export MC_HOST_arcadiatest=http://${s3_key}:${s3_secret}@127.0.0.1:9000
mc cp pkg/documentloaders/testdata/qa.csv arcadiatest/${bucket}/qa.csv
info "add tags to these files"
mc tag set arcadiatest/${bucket}/qa.csv "object_type=QA"

info "7.2 create dateset and versioneddataset and wait them ready"
kubectl apply -f config/samples/arcadia_v1alpha1_dataset.yaml
kubectl apply -f config/samples/arcadia_v1alpha1_versioneddataset.yaml
waitCRDStatusReady "VersionedDataset" "arcadia" "dataset-playground-v1"

info "7.3 create embedder and wait it ready"
kubectl apply -f config/samples/arcadia_v1alpha1_embedders.yaml
waitCRDStatusReady "Embedders" "arcadia" "zhipuai-embedders-sample"

info "7.4 create knowledgebase and wait it ready"
info "7.4.1 create knowledgebase based on chroma and wait it ready"
kubectl apply -f config/samples/arcadia_v1alpha1_knowledgebase.yaml
waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample"
sleep 3
info "7.4.2 create knowledgebase based on pgvector and wait it ready"
kubectl apply -f config/samples/arcadia_v1alpha1_knowledgebase_pgvector.yaml
waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample-pgvector"

info "7.5 check vectorstore has data"
info "7.5.1 check chroma vectorstore has data"
kubectl port-forward -n arcadia svc/arcadia-chromadb 8000:8000 >/dev/null 2>&1 &
chroma_pid=$!
info "port-forward chroma in pid: $chroma_pid"
sleep 3
collection_test_id=$(curl http://127.0.0.1:8000/api/v1/collections/arcadia_knowledgebase-sample | jq -r .id)
collection_test_count=$(curl http://127.0.0.1:8000/api/v1/collections/${collection_test_id}/count)
if [[ $collection_test_count =~ ^[0-9]+$ ]]; then
	info "collection test count: $collection_test_count"
else
	echo "$collection_test_count is not a number"
	exit 1
fi

info "7.5.2 check pgvector vectorstore has data"
kubectl port-forward -n arcadia svc/arcadia-postgresql 5432:5432 >/dev/null 2>&1 &
postgres_pid=$!
info "port-forward postgres in pid: $chroma_pid"
sleep 3
paasword=$(kubectl get secrets -n arcadia arcadia-postgresql -o json | jq -r '.data."postgres-password"' | base64 --decode)
if [[ $GITHUB_ACTIONS == "true" ]]; then
	docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h localhost -c "select document from langchain_pg_embedding;"
	pgdata=$(docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h localhost -c "select document from langchain_pg_embedding;")
else
	docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h host.docker.internal -c "select document from langchain_pg_embedding;"
	pgdata=$(docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h host.docker.internal -c "select document from langchain_pg_embedding;")
fi
if [[ -z $pgdata ]]; then
	info "get no data in postgres"
	exit 1
fi

info "7.6 update qa.csv to make sure it can be embedding"
echo "newquestion,newanswer,,," >>pkg/documentloaders/testdata/qa.csv
mc cp pkg/documentloaders/testdata/qa.csv arcadiatest/${bucket}/dataset/dataset-playground/v1/qa.csv
mc tag set arcadiatest/${bucket}/dataset/dataset-playground/v1/qa.csv "object_type=QA"
sleep 3
kubectl annotate knowledgebase/knowledgebase-sample-pgvector -n arcadia "arcadia.kubeagi.k8s.com.cn/update-source-file-time=$(date)"
sleep 3
waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample-pgvector"
if [[ $GITHUB_ACTIONS == "true" ]]; then
	docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h localhost -c "select document from langchain_pg_embedding;"
	pgdata=$(docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h localhost -c "select document from langchain_pg_embedding;")
else
	docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h host.docker.internal -c "select document from langchain_pg_embedding;"
	pgdata=$(docker run --net=host --entrypoint="" -e PGPASSWORD=$paasword kubeagi/postgresql:latest psql -U postgres -d arcadia -h host.docker.internal -c "select document from langchain_pg_embedding;")
fi
if [[ -z $pgdata ]]; then
	info "get no data in postgres"
	exit 1
else
	if [[ ! $pgdata =~ "newquestion" ]]; then
		info "get no new data in postgres"
		exit 1
	fi
fi

info "8 validate simple app can work normally"
info "Prepare dependent LLM service"
kubectl apply -f config/samples/app_shared_llm_service.yaml

info "8.1 app of llmchain"
kubectl apply -f config/samples/app_llmchain_englishteacher.yaml
waitCRDStatusReady "Application" "arcadia" "base-chat-english-teacher"
kubectl port-forward svc/arcadia-apiserver -n arcadia 8081:8081 >/dev/null 2>&1 &
portal_pid=$!
info "port-forward portal in pid: $portal_pid"
sleep 3
getRespInAppChat "base-chat-english-teacher" "arcadia" "hi how are you?" "" "true"

info "8.2 QA app using knowledgebase base"
info "8.2.1 QA app using knowledgebase base on chroma"
kubectl apply -f config/samples/app_retrievalqachain_knowledgebase.yaml
waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase"
sleep 3
getRespInAppChat "base-chat-with-knowledgebase" "arcadia" "旷工最小计算单位为多少天？" "" "true"

info "8.2.2 QA app using knowledgebase base on pgvector"
kubectl apply -f config/samples/app_retrievalqachain_knowledgebase_pgvector.yaml
waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase-pgvector"
sleep 3
getRespInAppChat "base-chat-with-knowledgebase" "arcadia" "旷工最小计算单位为多少天？" "" "true"

info "8.3 conversation chat app"
kubectl apply -f config/samples/app_llmchain_chat_with_bot.yaml
waitCRDStatusReady "Application" "arcadia" "base-chat-with-bot"
sleep 3
getRespInAppChat "base-chat-with-bot" "arcadia" "Hi I am Jim" "" "false"
getRespInAppChat "base-chat-with-bot" "arcadia" "What is my name?" ${resp_conversation_id} "false"
if [[ $resp != *"Jim"* ]]; then
	echo "Because conversationWindowSize is enabled to be 2, llm should record history, but resp:"$resp "dont contains Jim"
	exit 1
fi

info "8.4 check conversation list and message history"
curl -XPOST http://127.0.0.1:8081/chat/conversations --data '{"app_name": "base-chat-with-bot", "app_namespace": "arcadia"}'
data=$(jq -n --arg conversationID "$resp_conversation_id" '{"conversation_id":$conversationID, "app_name": "base-chat-with-bot", "app_namespace": "arcadia"}')
curl -XPOST http://127.0.0.1:8081/chat/messages --data "$data"
# There is uncertainty in the AI replies, most of the time, it will pass the test, a small percentage of the time, the AI will call names in each reply, causing the test to fail, therefore, temporarily disable the following tests
#getRespInAppChat "base-chat-with-bot" "arcadia" "What is your model?" ${resp_conversation_id} "false"
#getRespInAppChat "base-chat-with-bot" "arcadia" "Does your model based on gpt-3.5?" ${resp_conversation_id} "false"
#getRespInAppChat "base-chat-with-bot" "arcadia" "When was the model you used released?" ${resp_conversation_id} "false"
#getRespInAppChat "base-chat-with-bot" "arcadia" "What is my name?" ${resp_conversation_id} "false"
#if [[ $resp == *"Jim"* ]]; then
#	echo "Because conversationWindowSize is enabled to be 2, and current is the 6th conversation, llm should not record My name, but resp:"$resp "still contains Jim"
#	exit 1
#fi

info "8.5 apichain test"
kubectl apply -f config/samples/app_apichain_movie.yaml
waitCRDStatusReady "Application" "arcadia" "movie-bot"
sleep 3
curl -s -XPOST http://127.0.0.1:8081/chat --data '{"query":"年会不能停的主演有谁？","response_mode":"blocking","conversation_id":"","app_name":"movie-bot", "app_namespace":"arcadia"}'
#if [[ $resp != *"温度"* ]]; then
#	echo "Because conversationWindowSize is enabled to be 2, llm should record history, but resp:"$resp "dont contains Jim"
#	exit 1
#fi

info "9. show apiserver logs for debug"
kubectl logs --tail=100 -n arcadia -l app=arcadia-apiserver >/tmp/apiserver.log
cat /tmp/apiserver.log

info "all finished! ✅"
