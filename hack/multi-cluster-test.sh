#!/bin/bash
#
# Copyright 2023 The Primaza Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

TAG=$(git rev-parse --short HEAD)

CONTROLLER=primaza-controller:${TAG}
AGENTAPP=agentapp:${TAG}
AGENTSVC=agentsvc:${TAG}

MAIN_CLUSTER=primaza-main
WORKER_CLUSTER=primaza-worker
MAIN_KUBECONFIG=out/main-kubeconfig
WORKER_KUBECONFIG=out/worker-kubeconfig

if [[ ! -e bin/yq ]]; then
    make yq
fi

kind delete clusters ${MAIN_CLUSTER} ${WORKER_CLUSTER}

echo "creating main cluster" && \
    kind create cluster -q --name ${MAIN_CLUSTER} --kubeconfig ${MAIN_KUBECONFIG} && \
    bin/yq -i ".clusters[0].cluster.server = \"https://$(docker container inspect ${MAIN_CLUSTER}-control-plane | bin/yq '.[0].NetworkSettings.Networks.kind.IPAddress'):6443\"" ${MAIN_KUBECONFIG} &

echo "creating worker cluster" && \
    kind create cluster -q --name ${WORKER_CLUSTER} --kubeconfig ${WORKER_KUBECONFIG} && \
    bin/yq -i ".clusters[0].cluster.server = \"https://$(docker container inspect ${WORKER_CLUSTER}-control-plane | bin/yq '.[0].NetworkSettings.Networks.kind.IPAddress'):6443\"" ${WORKER_KUBECONFIG} &
wait

make primaza docker-build IMG="${CONTROLLER}"
make agentapp docker-build IMG="${AGENTAPP}"
make agentsvc docker-build IMG="${AGENTSVC}"

for name in $MAIN_CLUSTER $WORKER_CLUSTER; do
    for image in $CONTROLLER $AGENTAPP $AGENTSVC; do
        kind load docker-image "${image}" --name "${name}" &
    done
done
wait

make test-acceptance-wip \
    EXTRA_BEHAVE_ARGS="-k" \
    PRIMAZA_CONTROLLER_IMAGE_REF="${CONTROLLER}" \
    PRIMAZA_AGENTAPP_IMAGE_REF="${AGENTAPP}" \
    PRIMAZA_AGENTSVC_IMAGE_REF="${AGENTSVC}" \
    CLUSTER_PROVIDER=external \
    MAIN_KUBECONFIG="${MAIN_KUBECONFIG}" \
    WORKER_KUBECONFIG="${WORKER_KUBECONFIG}"
