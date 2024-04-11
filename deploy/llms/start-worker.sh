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
#         http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Start ray worker if configured
if [[ $RAY_ADDRESS != "" ]]; then
    echo "Run Ray worker..."
    ray start --address=$RAY_ADDRESS
    # wait for ray worker's resource to be available
    # TODO: maybe have better way to do this
    sleep 5
fi

echo "Run model worker..."
python3.9 -m $FASTCHAT_WORKER_NAME --model-names $FASTCHAT_REGISTRATION_MODEL_NAME \
    --model-path $FASTCHAT_MODEL_NAME_PATH --worker-address $FASTCHAT_WORKER_ADDRESS \
    --controller-address $FASTCHAT_CONTROLLER_ADDRESS \
    --num-gpus $NUMBER_GPUS  \
    --host 0.0.0.0 --port 21002 $SYSTEM_ARGS $EXTRA_ARGS
