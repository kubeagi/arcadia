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

function verify_datasource() {
    info "verify oss datasource"
    kubectl apply -f config/samples/arcadia_v1alpha1_datasource.yaml
    waitCRDStatusReady "Datasource" "arcadia" "datasource-sample"
    datasourceType=$(kubectl get datasource -n arcadia datasource-sample -o=jsonpath='{.metadata.labels.arcadia\.kubeagi\.k8s\.com\.cn/datasource-type}')
    if [[ $datasourceType != "oss" ]]; then
        error "Datasource should be oss but got $datasourceType"
        exit 1
    fi

    info "verify PostgreSQL datasource"
    kubectl apply -f config/samples/arcadia_v1alpha1_datasource_postgresql.yaml
    waitCRDStatusReady "Datasource" "arcadia" "datasource-postgresql-sample"
    datasourceType=$(kubectl get datasource -n arcadia datasource-postgresql-sample -o=jsonpath='{.metadata.labels.arcadia\.kubeagi\.k8s\.com\.cn/datasource-type}')
    if [[ $datasourceType != "postgresql" ]]; then
        error "Datasource should be oss but got $datasourceType"
        exit 1
    fi
}

function verify_vectorstore() {
    info "verify default vectorstore"
    waitCRDStatusReady "VectorStore" "arcadia" "arcadia-pgvector-vectorstore"
    info "verify PGVector vectorstore"
    kubectl apply -f config/samples/arcadia_v1alpha1_vectorstore_pgvector.yaml
    waitCRDStatusReady "VectorStore" "arcadia" "pgvector-sample"
}

function verify_dataset() {
    kubectl apply -f config/samples/arcadia_v1alpha1_dataset.yaml
    kubectl apply -f config/samples/arcadia_v1alpha1_versioneddataset.yaml
    waitCRDStatusReady "VersionedDataset" "arcadia" "dataset-playground-v1"
}