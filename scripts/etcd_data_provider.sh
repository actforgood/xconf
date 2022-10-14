#!/usr/bin/env bash

#
# This script provides the data used in integration tests for Etcd.
#
# Example of usage of this script:
# ./path/to/scripts/etcd_data_provider.sh
#


SCRIPT_PATH=$(dirname "$(readlink -f "$0")")

checkEtcdSaveKeyResponse() {
    out=$1
    if [ "$out" != "OK" ]; then
        printf '\033[0;31mFAIL\033[0m >>> %s' "$out"
        exit 1
    fi
}

KEYS=(json-key json-key/subkey yaml-key yaml-key/subkey plain-key plain-key/subkey)
DATA_FILES=(json-key json-subkey yaml-key yaml-subkey plain-key plain-subkey)
i=0
echo ">>> Provisioning Etcd keys..."
for key in "${KEYS[@]}"
do
	echo "${key}"
    value=$(cat "${SCRIPT_PATH}/../testdata/integration/${DATA_FILES[$i]}")
    out=$(docker exec xconf-etcd /bin/sh -c "export ETCDCTL_API=3 && /usr/local/bin/etcdctl put ${key} '${value}'" 2>&1)
    checkEtcdSaveKeyResponse "$out"
    out=$(docker exec xconf-etcds /bin/sh -c "export ETCDCTL_API=3 && /usr/local/bin/etcdctl --cacert=/certs/ca_cert.pem --endpoints=localhost:2389 put ${key} '${value}'" 2>&1)
    checkEtcdSaveKeyResponse "$out"
    i=$i+1
done
echo ">>> Successfully provisioned keys"
