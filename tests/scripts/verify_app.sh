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


function verify_app() {
    info "8.1 app of llmchain"
    kubectl apply -f config/samples/app_llmchain_englishteacher.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-english-teacher"
    EnableAPIServerPortForward
    sleep 3
    getRespInAppChat "base-chat-english-teacher" "arcadia" "hi how are you?" "" "true"

    info "8.2 QA app using knowledgebase base"
    info "8.2.1.1 QA app using knowledgebase base on chroma"
    kubectl apply -f config/samples/app_retrievalqachain_knowledgebase.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase"
    sleep 3
    getRespInAppChat "base-chat-with-knowledgebase" "arcadia" "公司的考勤管理制度适用于哪些人员？" "" "true"
    if [[ $ai_data != *"全体正式员工及实习生"* ]]; then
        echo "resp should contains '公司全体正式员工及实习生', but resp is:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-knowledgebase" "arcadia" "怀孕9个月以上每月可以享受几天假期？" "" "true"
    if [[ $ai_data != *"4"* ]]; then
        echo "resp should contains '4', but resp is:"$resp
        exit 1
    fi
    info "8.2.1.2 When no related doc is found, return application.spec.docNullReturn info, if set"
    getRespInAppChat "base-chat-with-knowledgebase" "arcadia" "飞天的主演是谁？" "" "true"
    expected=$(kubectl get applications -n arcadia base-chat-with-knowledgebase -o json | jq -r .spec.docNullReturn)
    if [[ $ai_data != $expected ]]; then
        echo "when no related doc is found, return application.spec.docNullReturn info should be:"$expected ", but resp:"$resp
        exit 1
    fi
    info "8.2.1.3 When no related doc is found and application.spec.docNullReturn is not set"
    kubectl patch applications -n arcadia base-chat-with-knowledgebase -p '{"spec":{"docNullReturn":""}}' --type='merge'
    getRespInAppChat "base-chat-with-knowledgebase" "arcadia" "飞天的主演是谁？" "" "true"

    info "8.2.2 QA app using knowledgebase base on pgvector"
    kubectl apply -f config/samples/app_retrievalqachain_knowledgebase_pgvector.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase-pgvector"
    sleep 3
    getRespInAppChat "base-chat-with-knowledgebase-pgvector" "arcadia" "公司的考勤管理制度适用于哪些人员？" "" "true"
    if [[ $ai_data != *"全体正式员工及实习生"* ]]; then
        echo "resp should contains '公司全体正式员工及实习生', but resp is:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-knowledgebase-pgvector" "arcadia" "怀孕9个月以上每月可以享受几天假期？" "" "true"
    if [[ $ai_data != *"4"* ]]; then
        echo "resp should contains '4', but resp is:"$resp
        exit 1
    fi
    info "8.2.2.2 When no related doc is found, return application.spec.docNullReturn info, if set"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector" "arcadia" "飞天的主演是谁？" "" "true"
    expected=$(kubectl get application -n arcadia base-chat-with-knowledgebase-pgvector -o json | jq -r .spec.docNullReturn)
    if [[ $ai_data != $expected ]]; then
        echo "when no related doc is found, return application.spec.docNullReturn info should be:"$expected ", but resp:"$resp
        exit 1
    fi
    info "8.2.2.3 When no related doc is found and application.spec.docNullReturn is not set"
    kubectl patch applications -n arcadia base-chat-with-knowledgebase-pgvector -p '{"spec":{"docNullReturn":""}}' --type='merge'
    getRespInAppChat "base-chat-with-knowledgebase-pgvector" "arcadia" "飞天的主演是谁？" "" "true"

    info "8.2.3 QA app using knowledgebase base on pgvector and rerank"
    kubectl apply -f config/samples/arcadia_v1alpha1_model_reranking_bce.yaml
    waitCRDStatusReady "Model" "arcadia" "bce-reranker"
    kubectl apply -f config/samples/arcadia_v1alpha1_worker_reranking_bce.yaml
    waitCRDStatusReady "Worker" "arcadia" "bce-reranker"
    kubectl apply -f config/samples/app_retrievalqachain_knowledgebase_pgvector_rerank.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase-pgvector-rerank"
    sleep 3
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank" "arcadia" "公司的考勤管理制度适用于哪些人员？" "" "true"
    if [[ $ai_data != *"全体正式员工及实习生"* ]]; then
        echo "resp should contains '公司全体正式员工及实习生', but resp is:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank" "arcadia" "怀孕9个月以上每月可以享受几天假期？" "" "true"
    if [[ $ai_data != *"4"* ]]; then
        echo "resp should contains '4', but resp is:"$resp
        exit 1
    fi
    info "8.2.3.2 When no related doc is found, return application.spec.docNullReturn info, if set"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank" "arcadia" "飞天的主演是谁？" "" "true"
    expected=$(kubectl get applications -n arcadia base-chat-with-knowledgebase-pgvector-rerank -o json | jq -r .spec.docNullReturn)
    if [[ $ai_data != $expected ]]; then
        echo "when no related doc is found, return application.spec.docNullReturn info should be:"$expected ", but resp:"$resp
        exit 1
    fi
    info "8.2.3.3 When no related doc is found and application.spec.docNullReturn is not set"
    kubectl patch applications -n arcadia base-chat-with-knowledgebase-pgvector-rerank -p '{"spec":{"docNullReturn":""}}' --type='merge'
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank" "arcadia" "飞天的主演是谁？" "" "true"

    info "8.2.4 QA app using knowledgebase base on pgvector and rerank and multiquery"
    kubectl apply -f config/samples/app_retrievalqachain_knowledgebase_pgvector_rerank_multiquery.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase-pgvector-rerank-multiquery"
    sleep 3
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank-multiquery" "arcadia" "公司的考勤管理制度适用于哪些人员？" "" "true"
    if [[ $ai_data != *"全体正式员工及实习生"* ]]; then
        echo "resp should contains '公司全体正式员工及实习生', but resp is:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank-multiquery" "arcadia" "怀孕9个月以上每月可以享受几天假期？" "" "true"
    if [[ $ai_data != *"4"* ]]; then
        echo "resp should contains '4', but resp is:"$resp
        exit 1
    fi
    info "8.2.4.2 When no related doc is found, return application.spec.docNullReturn info, if set"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank-multiquery" "arcadia" "飞天的主演是谁？" "" "true"
    expected=$(kubectl get applications -n arcadia base-chat-with-knowledgebase-pgvector-rerank-multiquery -o json | jq -r .spec.docNullReturn)
    if [[ $ai_data != $expected ]]; then
        echo "when no related doc is found, return application.spec.docNullReturn info should be:"$expected ", but resp:"$resp
        exit 1
    fi
    info "8.2.4.3 When no related doc is found and application.spec.docNullReturn is not set"
    kubectl patch applications -n arcadia base-chat-with-knowledgebase-pgvector-rerank-multiquery -p '{"spec":{"docNullReturn":""}}' --type='merge'
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-rerank-multiquery" "arcadia" "飞天的主演是谁？" "" "true"

    info "8.2.5 QA app using knowledgebase base on pgvector and multiquery"
    kubectl apply -f config/samples/app_retrievalqachain_knowledgebase_pgvector_multiquery.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase-pgvector-multiquery"
    sleep 3
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-multiquery" "arcadia" "公司的考勤管理制度适用于哪些人员？" "" "true"
    if [[ $ai_data != *"全体正式员工及实习生"* ]]; then
        echo "resp should contains '公司全体正式员工及实习生', but resp is:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-multiquery" "arcadia" "怀孕9个月以上每月可以享受几天假期？" "" "true"
    if [[ $ai_data != *"4"* ]]; then
        echo "resp should contains '4', but resp is:"$resp
        exit 1
    fi
    info "8.2.5.2 When no related doc is found, return application.spec.docNullReturn info, if set"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-multiquery" "arcadia" "飞天的主演是谁？" "" "true"
    expected=$(kubectl get applications -n arcadia base-chat-with-knowledgebase-pgvector-multiquery -o json | jq -r .spec.docNullReturn)
    if [[ $ai_data != $expected ]]; then
        echo "when no related doc is found, return application.spec.docNullReturn info should be:"$expected ", but resp:"$resp
        exit 1
    fi
    info "8.2.5.3 When no related doc is found and application.spec.docNullReturn is not set"
    kubectl patch applications -n arcadia base-chat-with-knowledgebase-pgvector-multiquery -p '{"spec":{"docNullReturn":""}}' --type='merge'
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-multiquery" "arcadia" "飞天的主演是谁？" "" "true"

    info "8.3 conversation chat app"
    kubectl apply -f config/samples/app_llmchain_chat_with_bot.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-bot"
    sleep 3
    getRespInAppChat "base-chat-with-bot" "arcadia" "Hi I am Bob" "" "false"
    getRespInAppChat "base-chat-with-bot" "arcadia" "Hi I am Jim" "" "false"
    getRespInAppChat "base-chat-with-bot" "arcadia" "What is my name?" ${resp_conversation_id} "false"
    if [[ $resp != *"Jim"* ]]; then
        echo "Because conversationWindowSize is enabled to be 2, llm should record history, but resp:"$resp "dont contains Jim"
        exit 1
    fi

    info "8.4 check other chat rest api"
    info "8.4.1 conversation list"
    resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/conversations --data '{"app_name": "base-chat-with-bot", "app_namespace": "arcadia"}')
    echo $resp | jq .
    icon=$(echo $resp | jq -r '.[0].icon')
    if [[ $icon == "null" ]] || [[ -z $icon ]]; then
        echo "should has icon."
        exit 1
    fi
    delete_conversation_id=$(echo $resp | jq -r '.[0].id')
    info "8.4.2 message list"
    data=$(jq -n --arg conversationID "$delete_conversation_id" '{"conversation_id":$conversationID, "app_name": "base-chat-with-bot", "app_namespace": "arcadia"}')
    resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/messages --data "$data")
    echo $resp | jq .
    info "8.4.3 message references"
    resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/conversations --data '{"app_name": "base-chat-with-knowledgebase-pgvector", "app_namespace": "arcadia"}')
    message_id=$(echo $resp | jq -r '.[1].messages[0].id')
    conversation_id=$(echo $resp | jq -r '.[1].id')
    data=$(jq -n --arg conversationID "$conversation_id" '{"conversation_id":$conversationID, "app_name": "base-chat-with-knowledgebase-pgvector", "app_namespace": "arcadia"}')
    resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/messages/$message_id/references --data "$data")
    echo $resp | jq .
    info "8.4.4 delete conversation"
    resp=$(curl --max-time $TimeoutSeconds -s --show-error -XDELETE http://127.0.0.1:8081/chat/conversations/$delete_conversation_id)
    echo $resp | jq .
    resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/conversations --data '{"app_name": "base-chat-with-bot", "app_namespace": "arcadia"}')
    if [[ $resp == *"$delete_conversation_id"* ]]; then
        echo "delete conversation failed"
        exit 1
    fi
    info "8.4.5 get app prompt starters"
    attempt=0
    while true; do
        info "sleep 3 seconds"
        sleep 3
        info "get app prompt starters without knowledgebase"
        resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/prompt-starter --data '{"app_name": "base-chat-with-bot"}' -H 'namespace: arcadia')
        echo $resp | jq .
        if [[ $resp == *"error"* ]]; then
            attempt=$((attempt + 1))
            if [ $attempt -gt $RETRY_COUNT ]; then
                echo "❌: Failed. Retry count exceeded."
                exit 1
            fi
            echo "🔄: Failed. Attempt $attempt/$RETRY_COUNT"
            kill $portal_pid >/dev/null 2>&1
            EnableAPIServerPortForward
            if [[ $resp == *"googleapi: Error"* ]]; then
                echo "google api error, will retry after 60s"
                sleep 60
            fi
            continue
        fi
        info "get app prompt starters with knowledgebase"
        resp=$(curl --max-time $TimeoutSeconds -s --show-error -XPOST http://127.0.0.1:8081/chat/prompt-starter --data '{"app_name": "base-chat-with-knowledgebase-pgvector"}' -H 'namespace: arcadia')
        echo $resp | jq .
        if [[ $resp == *"error"* ]]; then
            echo "failed"
            exit 1
        fi
        break
    done

    info "8.4.6 chat with document"
    kubectl apply -f config/samples/app_llmchain_abstract.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-document-assistant"
    fileUploadSummarise "base-chat-document-assistant" "arcadia" "./pkg/documentloaders/testdata/arcadia-readme.pdf"
    getRespInAppChat "base-chat-document-assistant" "arcadia" "what is arcadia?" ${resp_conversation_id} "false"
    getRespInAppChat "base-chat-document-assistant" "arcadia" "Does your model based on gpt-3.5?" ${resp_conversation_id} "false"

    info "8.4.7 chat with document with knowledgebase"
    fileUploadSummarise "base-chat-with-knowledgebase-pgvector" "arcadia" "./pkg/documentloaders/testdata/arcadia-readme.pdf"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector" "arcadia" "what is arcadia?" ${resp_conversation_id} "false"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector" "arcadia" "公司的考勤管理制度适用于哪些人员？" ${resp_conversation_id} "false"


    info "8.5 apichain test"
    kubectl apply -f config/samples/app_apichain_movie.yaml
    waitCRDStatusReady "Application" "arcadia" "movie-bot"
    sleep 3
    getRespInAppChat "movie-bot" "arcadia" "年会不能停的主演是谁？" "" "false"
    #if [[ $resp != *"温度"* ]]; then
    #	echo "Because conversationWindowSize is enabled to be 2, llm should record history, but resp:"$resp "dont contains Jim"
    #	exit 1
    #fi
    #if [[ $GITHUB_ACTIONS != "true" ]]; then
    info "8.6 tool test"
    kubectl apply -f config/samples/app_llmchain_chat_with_bot_tool.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-bot-tool"
    sleep 3
    info "8.6.1 conversation test"
    info "23*34 结果应该是 782, 结果再乘2是 1564, 再减去564是 1000"
    getRespInAppChat "base-chat-with-bot-tool" "arcadia" "计算 23*34 的结果" "" "false"
    if [[ $ai_data != *"782"* ]]; then
        echo "resp should contains 782, but resp:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-bot-tool" "arcadia" "结果再乘2" ${resp_conversation_id} "false"
    if [[ $ai_data != *"1564"* ]]; then
        echo "resp should contains 1564, but resp:"$resp
        exit 1
    fi
    getRespInAppChat "base-chat-with-bot-tool" "arcadia" "结果再减去564" ${resp_conversation_id} "false"
    if [[ $ai_data != *"1000"* ]]; then
        echo "resp should contains 1000, but resp:"$resp
        exit 1
    fi
    #	info "8.6.1 bingsearch test"
    #	getRespInAppChat "base-chat-with-bot-tool" "arcadia" "用30字介绍一下云原生" "" "true"
    #	if [ -z "$references" ] || [ "$references" = "null" ]; then
    #		echo $resp
    #		exit 1
    #	fi
    sleep 3
    info "8.6.2 calculator test"
    info "23*34 结果应该是 782"
    getRespInAppChat "base-chat-with-bot-tool" "arcadia" "计算 23*34 的结果" "" "true"
    if [[ $ai_data != *"782"* ]]; then
        echo "resp should contains 782, but resp:"$resp
        exit 1
    fi
    sleep 3
    info "8.6.3 webpage test"
    info "说的是 kubeedge 在 cmcc 上的使用情况"
    getRespInAppChat "base-chat-with-bot-tool" "arcadia" "https://kubeedge.io/zh/case-studies/CMCC-10086 简单总结一下说了什么" "" "true"
    sleep 3
    info "8.6.4 weather test"
    info "说的是北京今天的天气情况"
    getRespInAppChat "base-chat-with-bot-tool" "arcadia" "北京今天的天气如何？" "" "true"

    info "8.7 tool test with knowledgebase and qachain"
    kubectl apply -f config/samples/app_retrievalqachain_knowledgebase_pgvector_tool.yaml
    waitCRDStatusReady "Application" "arcadia" "base-chat-with-knowledgebase-pgvector-tool"
    kubectl patch KnowledgeBaseRetriever -n arcadia base-chat-with-knowledgebase -p '{"spec":{"scoreThreshold":0.9}}' --type='merge'
    sleep 3
    #	info "8.7.1 bingsearch test"
    #	getRespInAppChat "base-chat-with-knowledgebase-pgvector-tool" "arcadia" "用30字介绍一下云原生" "" "true"
    #	if [ -z "$references" ] || [ "$references" = "null" ]; then
    #		echo $resp
    #		exit 1
    #	fi
    sleep 3
    info "8.7.2 calculator test"
    info "23*35 结果应该是 805"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-tool" "arcadia" "计算 23*35 的结果" "" "true"
    if [[ $ai_data != *"805"* ]]; then
        echo "resp should contains 805, but resp:"$resp
        exit 1
    fi
    sleep 3
    info "8.7.3 webpage test"
    info "说的是 kubeedge 在 cmcc 上的使用情况"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-tool" "arcadia" "https://kubeedge.io/zh/case-studies/CMCC-10086 简单总结一下说了什么" "" "true"
    sleep 3
    info "8.7.4 weather test"
    info "说的是北京今天的天气情况"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-tool" "arcadia" "北京今天的天气如何？" "" "true"
    sleep 3
    info "8.7.5 knowledgebase test"
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-tool" "arcadia" "公司的考勤管理制度适用于哪些人员？" "" "true"
    if [[ $ai_data != *"全体正式员工及实习生"* ]]; then
        echo "resp should contains '公司全体正式员工及实习生', but resp is:"$resp
        exit 1
    fi
    # 0.9 is too high for chunk text segmentation files
    kubectl patch KnowledgeBaseRetriever -n arcadia base-chat-with-knowledgebase -p '{"spec":{"scoreThreshold":0.5}}' --type='merge'
    getRespInAppChat "base-chat-with-knowledgebase-pgvector-tool" "arcadia" "怀孕9个月以上每月可以享受几天假期？" "" "true"
    if [[ $ai_data != *"4"* ]]; then
        echo "resp should contains '4', but resp is:"$resp
        exit 1
    fi
    #fi
}