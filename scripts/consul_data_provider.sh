#!/usr/bin/env bash

#
# This script provides the data used in integration tests for Consul.
#
# Example of usage of this script:
# ./path/to/scripts/consul_data_provider.sh
# or
# ./path/to/scripts/consul_data_provider.sh http://consul.example.com:8500
#

CONSUL=http://127.0.0.1:8500
if [ "$CONSUL_HTTP_ADDR" != "" ]; then
    CONSUL="http://$CONSUL_HTTP_ADDR"
fi
if [ "$1" != "" ]; then
    CONSUL=$1
fi

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")

checkConsulSaveKeyResponse() {
    out=$1
    if [ "$out" != "true" ]; then
        printf '\033[0;31mFAIL\033[0m >>> %s' "$out"
        exit 1
    fi
}

KEYS=(json-key json-key/subkey yaml-key yaml-key/subkey plain-key plain-key/subkey)
DATA_FILES=(json-key json-subkey yaml-key yaml-subkey plain-key plain-subkey)
i=0
echo ">>> Provisioning Consul keys..."
for key in "${KEYS[@]}"
do
	echo "${key}"
    out=$(curl -s -S -X PUT --data-binary "@${SCRIPT_PATH}/../testdata/integration/${DATA_FILES[$i]}" "${CONSUL}/v1/kv/${key}" 2>&1)
    checkConsulSaveKeyResponse "$out"
    i=$i+1
done
echo ">>> Successfully provisioned keys"
