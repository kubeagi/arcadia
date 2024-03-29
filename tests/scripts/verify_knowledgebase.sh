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

function verify_knowledgebase() {
    info "create knowledgebase based on chroma and wait it ready"
    kubectl apply -f config/samples/arcadia_v1alpha1_knowledgebase.yaml
    waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample"
    sleep 3
    info "check chroma vectorstore has data"
    collection_test_id=$(curl --max-time $TimeoutSeconds http://127.0.0.1:8000/api/v1/collections/arcadia_knowledgebase-sample | jq -r .id)
    collection_test_count=$(curl --max-time $TimeoutSeconds http://127.0.0.1:8000/api/v1/collections/${collection_test_id}/count)
    if [[ $collection_test_count =~ ^[0-9]+$ ]]; then
        info "collection test count: $collection_test_count"
    else
        echo "$collection_test_count is not a number"
        exit 1
    fi

    info "create knowledgebase based on pgvector and wait it ready"
    kubectl apply -f config/samples/arcadia_v1alpha1_knowledgebase_pgvector.yaml
    waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample-pgvector"
    sleep 3
    info "check pgvector vectorstore has data"
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
}

function verify_knowledgebase_force_reconcile() {
    info "update and upload new data"
    echo "newquestion,newanswer,,," >>pkg/documentloaders/testdata/qa.csv
    mc cp pkg/documentloaders/testdata/qa.csv arcadiatest/${bucket}/dataset/dataset-playground/v1/qa.csv
    mc tag set arcadiatest/${bucket}/dataset/dataset-playground/v1/qa.csv "object_type=QA"
    sleep 3
    
    info "update annotation to force reconcile"
    kubectl annotate knowledgebase/knowledgebase-sample-pgvector -n arcadia "arcadia.kubeagi.k8s.com.cn/update-source-file-time=$(date)"
    sleep 3
    waitCRDStatusReady "KnowledgeBase" "arcadia" "knowledgebase-sample-pgvector"

    info "check pgvector vectorstore has the new data"
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
}