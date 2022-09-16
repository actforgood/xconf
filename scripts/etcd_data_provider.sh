#!/usr/bin/env bash

#
# This script provides the data used in integration tests for Etcd.
#
# Example of running a local Etcd instance (https://etcd.io/docs/v3.5/op-guide/container/): 
# docker run -d --name=integration-etcd -p 2379:2379 quay.io/coreos/etcd:v3.5.5 /usr/local/bin/etcd -advertise-client-urls http://integration-etcd:2379 -listen-client-urls http://0.0.0.0:2379
#
# Example of usage of this script:
# ./path/to/scripts/etcd_data_provider.sh
#


SCRIPT_PATH=$(dirname "$(readlink -f "$0")")

checkEtcdSaveKeyResponse() {
    out=$1
    if [ "$out" != "OK" ]; then
        echo "Failed: $@"
        exit 1
    fi
}

KEYS=(json-key json-key/subkey yaml-key yaml-key/subkey plain-key plain-key/subkey)
DATA_FILES=(json-key json-subkey yaml-key yaml-subkey plain-key plain-subkey)
i=0
for key in "${KEYS[@]}"
do
	echo "${key}"
    value=$(cat "${SCRIPT_PATH}/../testdata/integration/${DATA_FILES[$i]}")
    out=$(docker exec integration-etcd /bin/sh -c "export ETCDCTL_API=3 && /usr/local/bin/etcdctl put ${key} '${value}'")
    checkEtcdSaveKeyResponse "$out"
    i=$i+1
done
