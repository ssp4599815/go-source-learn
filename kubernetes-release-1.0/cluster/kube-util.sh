#!/bin/bash

# Copyright 2014 The Kubernetes Authors All rights reserved.
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

# A library of helper functions that each provider hosting Kubernetes must implement to use cluster/kube-*.sh scripts.

# Must ensure that the following ENV vars are set
function detect-master {
	echo "KUBE_MASTER_IP: $KUBE_MASTER_IP"
	echo "KUBE_MASTER: $KUBE_MASTER"
}

# Get minion names if they are not static.
function detect-minion-names {
        echo "MINION_NAMES: ${MINION_NAMES[*]}"
}

# Get minion IP addresses and store in KUBE_MINION_IP_ADDRESSES[]
function detect-minions {
	echo "KUBE_MINION_IP_ADDRESSES=[]"
}

# Verify prereqs on host machine
function verify-prereqs {
	echo "TODO"
}

# Instantiate a kubernetes cluster
function kube-up {
	echo "TODO"
}

# Delete a kubernetes cluster
function kube-down {
	echo "TODO"
}

# Update a kubernetes cluster
function kube-push {
	echo "TODO"
}

# Prepare update a kubernetes component
function prepare-push {
	echo "TODO"
}

# Update a kubernetes master
function push-master {
	echo "TODO"
}

# Update a kubernetes node
function push-node {
	echo "TODO"
}

# Execute prior to running tests to build a release if required for env
function test-build-release {
	echo "TODO"
}

# Execute prior to running tests to initialize required structure
function test-setup {
	echo "TODO"
}

# Execute after running tests to perform any required clean-up
function test-teardown {
	echo "TODO: test-teardown" 1>&2
}

# Providers util.sh scripts should define functions that override the above default functions impls
if [ -n "${KUBERNETES_PROVIDER}" ]; then
	PROVIDER_UTILS="${KUBE_ROOT}/cluster/${KUBERNETES_PROVIDER}/util.sh"
	if [ -f ${PROVIDER_UTILS} ]; then
		source "${PROVIDER_UTILS}"
	fi
fi
