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

KindName="kubeagi"
HelmTimeout=${HelmTimeout:-"1800s"}

function setup_testenv() {
	info "create kind cluster"
	make kind
	df -h

	info "load arcadia image to kind"
	docker tag controller:latest controller:example-e2e
	kind load docker-image controller:example-e2e --name=$KindName

	info "install arcadia"
	kubectl create namespace arcadia
	helm install -narcadia arcadia deploy/charts/arcadia -f tests/deploy-values.yaml \
		--set controller.image=controller:example-e2e --set apiserver.image=controller:example-e2e \
		--wait --timeout $HelmTimeout

	info "check system datasource arcadia-minio(system datasource)"
	waitCRDStatusReady "Datasource" "arcadia" "arcadia-minio"

	kubectl port-forward -n arcadia svc/arcadia-minio 9000:9000 >/dev/null 2>&1 &
	minio_pid=$!
	info "port-forward minio in pid: $minio_pid"

	kubectl port-forward -n arcadia svc/arcadia-chromadb 8000:8000 >/dev/null 2>&1 &
	chroma_pid=$!
	info "port-forward chroma in pid: $chroma_pid"

	kubectl port-forward -n arcadia svc/arcadia-postgresql 5432:5432 >/dev/null 2>&1 &
	postgres_pid=$!
	info "port-forward postgres in pid: $postgres_pid"

	sleep 3
	info "upload some test file to system datasource"
	bucket=$(kubectl get datasource -n arcadia datasource-sample -o json | jq -r .spec.oss.bucket)
	s3_key=$(kubectl get secrets -n arcadia datasource-sample-authsecret -o json | jq -r ".data.rootUser" | base64 --decode)
	s3_secret=$(kubectl get secrets -n arcadia datasource-sample-authsecret -o json | jq -r ".data.rootPassword" | base64 --decode)
	export MC_HOST_arcadiatest=http://${s3_key}:${s3_secret}@127.0.0.1:9000
	mc cp pkg/documentloaders/testdata/qa.csv arcadiatest/${bucket}/qa.csv
	mc cp pkg/documentloaders/testdata/chunk.csv arcadiatest/${bucket}/chunk.csv

	info "add tags to these files"
	mc tag set arcadiatest/${bucket}/qa.csv "object_type=QA"
	mc tag set arcadiatest/${bucket}/chunk.csv "object_type=QA"

	info "prepare shared llm service"
	if [[ $GITHUB_ACTIONS == "true" ]]; then
		info "in github action, use gemini"
		sed -i 's/model: chatglm_turbo/model: gemini-pro/g' config/samples/*
		sed -i 's/model: glm-4/model: gemini-pro/g' config/samples/*
		case "$GITHUB_ACTION_NO" in
		1)
			info "in github action no 1, use gemini apikey github-action-1"
			sed -i 's/apiKey: "QUl6YVN5QVZOdGRYOHpkeU5pNWpubzNYSExUWGM0UnpJSGxIRUFz"/apiKey: "QUl6YVN5QTBBWGVNOEJoRGpoSDN3MjBYdHc3NEQ3QUpVaV9meFRr"/g' config/samples/app_shared_llm_service_gemini.yaml
			;;
		2)
			info "in github action no 2, use gemini apikey github-action-2"
			sed -i 's/apiKey: "QUl6YVN5QVZOdGRYOHpkeU5pNWpubzNYSExUWGM0UnpJSGxIRUFz"/apiKey: "QUl6YVN5QlZPeXpQUlc0aE5tQ244QkV1MmxBcEYyeWo2eVVfcU93"/g' config/samples/app_shared_llm_service_gemini.yaml
			;;
		3)
			info "in github action no 3, use gemini apikey github-action-3"
			sed -i 's/apiKey: "QUl6YVN5QVZOdGRYOHpkeU5pNWpubzNYSExUWGM0UnpJSGxIRUFz"/apiKey: "QUl6YVN5RHJlSmtPZXZXZHZ5NGRUU1lrbGFFTFVzN0tQQktUZXdZ"/g' config/samples/app_shared_llm_service_gemini.yaml
			;;
		*) ;;
		esac
		kubectl apply -f config/samples/app_shared_llm_service_gemini.yaml
	else
		info "in local, use zhipu"
		kubectl apply -f config/samples/app_shared_llm_service_zhipu.yaml
	fi
}

function teardown_testenv() {
	info "delete arcadia"
	helm uninstall -narcadia arcadia
	kubectl delete namespace arcadia
	df -h
}


