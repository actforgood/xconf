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
printf "\033[0;34m>>> Provisioning Etcd keys...\033[0m\n"
for key in "${KEYS[@]}"
do
	echo "${key}"
    value=$(cat "${SCRIPT_PATH}/../testdata/integration/${DATA_FILES[$i]}")
    out=$(docker exec xconf-etcd /usr/local/bin/etcdctl put "${key}" "${value}" 2>&1)
    checkEtcdSaveKeyResponse "$out"
    out=$(docker exec xconf-etcds /usr/local/bin/etcdctl --cacert=/certs/ca_cert.pem --endpoints=localhost:2389 put "${key}" "${value}" 2>&1)
    checkEtcdSaveKeyResponse "$out"
    i=$i+1
done
printf "\033[0;34m>>> Successfully provisioned keys\033[0m\n"
