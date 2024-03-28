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


####################################################################################
# Following functions are for debugging purpose   #
####################################################################################

TimeoutSeconds=${TimeoutSeconds:-"300"}

function debugInfo {
	if [[ $? -eq 0 ]]; then
		exit 0
	fi
	if [[ $debug -ne 0 ]]; then
		exit 1
	fi
	if [[ $GITHUB_ACTIONS == "true" ]]; then
		warning "debugInfo start 🧐"
		mkdir -p $LOG_DIR || true
		df -h

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


####################################################################################
# Following functions are used to check pod status,CRD status,chat status,etc...   #
####################################################################################

function waitPodReady() {
	namespace=$1
	podLabel=$2
	START_TIME=$(date +%s)
	while true; do
		readStatus=$(kubectl -n${namespace} get po -l ${podLabel} --ignore-not-found=true -o json | jq -r '.items[0].status.conditions[] | select(."type"=="Ready") | .status')
		if [[ $readStatus == "True" ]]; then
			info "Pod:${podLabel} ready"
			break
		fi
		kubectl -n${namespace} get po -l ${podLabel}

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

function EnableAPIServerPortForward() {
	waitPodReady "arcadia" "app=arcadia-apiserver"
	if [ $portal_pid -ne 0 ]; then
		kill $portal_pid >/dev/null 2>&1
	fi
	echo "re port-forward apiserver..."
	kubectl port-forward svc/arcadia-apiserver -n arcadia 8081:8081 >/dev/null 2>&1 &
	portal_pid=$!
	sleep 3
	info "port-forward apiserver in pid: $portal_pid"
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
		if [[ ${source} == "Worker" ]]; then
			if [ $ELAPSED_TIME -gt 1800 ]; then
				error "Timeout reached"
				exit 1
			fi
		else
			if [ $ELAPSED_TIME -gt $TimeoutSeconds ]; then
				error "Timeout reached"
				exit 1
			fi
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
	attempt=0
	while true; do
		info "sleep 3 seconds"
		sleep 3
		data=$(jq -n --arg appname "$appname" --arg query "$query" --arg conversationID "$conversationID" '{"query":$query,"response_mode":"blocking","conversation_id":$conversationID,"app_name":$appname}')
		resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat --data "$data" -H "namespace: ${namespace}")
		ai_data=$(echo $resp | jq -r '.message')
		references=$(echo $resp | jq -r '.references')
		if [ -z "$ai_data" ] || [ "$ai_data" = "null" ]; then
			echo $resp
			EnableAPIServerPortForward
			if [[ $resp == *"googleapi: Error"* ]]; then
				echo "google api error, will retry after 60s"
				sleep 60
			fi
			attempt=$((attempt + 1))
			if [ $attempt -gt $RETRY_COUNT ]; then
				echo "❌: Failed. Retry count exceeded."
				exit 1
			fi
			echo "🔄: Failed. Attempt $attempt/$RETRY_COUNT"
			continue
		fi
		echo "👤: ${query}"
		echo "🤖: ${ai_data}"
		echo "🔗: ${references}"
		break
	done
	resp_conversation_id=$(echo $resp | jq -r '.conversation_id')

	if [ $testStream == "true" ]; then
		attempt=0
		while true; do
			info "sleep 5 seconds"
			sleep 5
			info "just test stream mode"
			data=$(jq -n --arg appname "$appname" --arg query "$query" --arg conversationID "$conversationID" '{"query":$query,"response_mode":"streaming","conversation_id":$conversationID,"app_name":$appname}')
			curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat --data "$data" -H "namespace: ${namespace}"
			if [[ $? -ne 0 ]]; then
				attempt=$((attempt + 1))
				if [ $attempt -gt $RETRY_COUNT ]; then
					echo "❌: Failed. Retry count exceeded."
					exit 1
				fi
				echo "🔄: Failed. Attempt $attempt/$RETRY_COUNT"
				EnableAPIServerPortForward
				echo "and wait 60s for google api error"
				sleep 60
				continue
			fi
			break
		done
	fi
}

function fileUploadSummarise() {
	appname=$1
	namespace=$2
	filename=$3
	attempt=0
	while true; do
		info "sleep 3 seconds"
		sleep 3
		resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST --form file=@$filename --form app_name=$appname -H "namespace: ${namespace}" -H "Content-Type: multipart/form-data" http://127.0.0.1:8081/chat/conversations/file)
		doc_data=$(echo $resp | jq -r '.document')
		if [ -z "$doc_data" ]; then
			echo $resp
			EnableAPIServerPortForward
			if [[ $resp == *"googleapi: Error"* ]]; then
				echo "google api error, will retry after 60s"
				sleep 60
			fi
			attempt=$((attempt + 1))
			if [ $attempt -gt $RETRY_COUNT ]; then
				echo "❌: Failed. Retry count exceeded."
				exit 1
			fi
			echo "🔄: Failed. Attempt $attempt/$RETRY_COUNT"
			continue
		fi
		echo "👤: ${filename}"
		echo "🤖: ${doc_data}"
		break
	done
	file_id=$(echo $resp | jq -r '.document.object')
	resp_conversation_id=$(echo $resp | jq -r '.conversation_id')
	attempt=0
	while true; do
		info "sleep 3 seconds to sumerize doc"
		sleep 3
		data=$(jq -n --arg fileid "$file_id" --arg appname "$appname" --arg query "总结一下" --arg conversationID "$resp_conversation_id" '{"query":$query,"response_mode":"blocking","conversation_id":$conversationID,"app_name":$appname, "files": [$fileid]}')
		resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat --data "$data" -H "namespace: ${namespace}")
		ai_data=$(echo $resp | jq -r '.message')
		references=$(echo $resp | jq -r '.references')
		if [ -z "$ai_data" ] || [ "$ai_data" = "null" ]; then
			echo $resp
			EnableAPIServerPortForward
			if [[ $resp == *"googleapi: Error"* ]]; then
				echo "google api error, will retry after 60s"
				sleep 60
			fi
			attempt=$((attempt + 1))
			if [ $attempt -gt $RETRY_COUNT ]; then
				echo "❌: Failed. Retry count exceeded."
				exit 1
			fi
			echo "🔄: Failed. Attempt $attempt/$RETRY_COUNT"
			continue
		fi
		echo "👤: 总结一下"
		echo "🤖: ${ai_data}"
		echo "🔗: ${references}"
		break
	done
	resp_conversation_id=$(echo $resp | jq -r '.conversation_id')

	if [ $testStream == "true" ]; then
		attempt=0
		while true; do
			info "sleep 5 seconds"
			sleep 5
			info "just test stream mode"
			data=$(jq -n --arg fileid "$file_id" --arg appname "$appname" --arg query "总结一下" --arg conversationID "$resp_conversation_id" '{"query":$query,"response_mode":"blocking","conversation_id":$conversationID,"app_name":$appname, "files": [$fileid]}')
			curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat --data "$data" -H "namespace: ${namespace}"
			if [[ $? -ne 0 ]]; then
				attempt=$((attempt + 1))
				if [ $attempt -gt $RETRY_COUNT ]; then
					echo "❌: Failed. Retry count exceeded."
					exit 1
				fi
				echo "🔄: Failed. Attempt $attempt/$RETRY_COUNT"
				EnableAPIServerPortForward
				echo "and wait 60s for google api error"
				sleep 60
				continue
			fi
			break
		done
	fi
}
