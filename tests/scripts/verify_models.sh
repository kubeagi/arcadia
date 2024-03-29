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

# TODO gemini embedding not support chinese now https://github.com/kubeagi/arcadia/issues/739#issuecomment-1960679242
#if [[ $GITHUB_ACTIONS == "true" ]]; then
#	info "in github action, use gemini"
#	kubectl apply -f config/samples/arcadia_v1alpha1_embedders_gemini.yaml
#else
#	info "in local, use zhipu"
#	kubectl apply -f config/samples/arcadia_v1alpha1_embedders_zhipu.yaml
#fi

source ./scripts/utils.sh

function verify_3rd_party_embedder() {
    kubectl apply -f config/samples/arcadia_v1alpha1_embedders_zhipu.yaml
    waitCRDStatusReady "Embedders" "arcadia" "embedders-sample"

    set_system_embedder arcadia-embedder embedders-sample
}

function set_system_embedder() {
    old_embedder=$1
    new_embedder=$2
    kubectl get cm -n arcadia arcadia-config -o yaml | sed -e "s/${old_embedder}/${new_embedder}/g" | kubectl apply -f -
    kubectl delete po -n arcadia -l control-plane=arcadia-arcadia
    kubectl delete po -n arcadia -l app=arcadia-apiserver
    waitPodReady "arcadia" "control-plane=arcadia-arcadia"
    waitPodReady "arcadia" "app=arcadia-apiserver"
}