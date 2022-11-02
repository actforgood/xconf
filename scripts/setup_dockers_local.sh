#!/usr/bin/env bash

#
# This script brings up Consul, Etcd, Etcd with TLS containers.
#
# Example of usage of this script:
# ./path/to/scripts/etcd_data_provider.sh
#

local=${GITHUB_WORKFLOW:-local}
if [ "$local" != "local" ]; then
    exit 0 # we're not locally
fi

DOCKER_CONSUL_IMAGE="consul:1.13.3"
DOCKER_ETCD_IMAGE="quay.io/coreos/etcd:v3.5.5"
SCRIPT_PATH=$(dirname "$(readlink -f "$0")")

# setUpLocalDocker ensures docker containers are up and running.
setUpLocalDocker() {
    container=$1
    existing=$(docker container ls -a | awk '{print $NF}' | grep -E "^${container}$")
    if [ "$existing" != "" ]; then
        running=$(docker ps | awk '{print $NF}' | grep -E "^${container}$")
        if [ "$running" != "" ]; then
            printf "\033[0;34m>>> %s is up and running\033[0m\n" "$container"
            checkIsHealthy "$container"
            return
        else
            printf "\033[0;34m>>> %s is stopped, resurrecting it...\033[0m\n" "$container"
            docker rm "$container"
            dockerRun "$container"
            checkIsHealthy "$container"
        fi
    else
        printf "\033[0;34m>>> %s not found, bringing it up...\033[0m\n" "$container"
        dockerPull "$container"
        dockerRun "$container"
        checkIsHealthy "$container"
    fi
}

# dockerRun runs Consul / Etcd / Etcd with TLS.
dockerRun() {
    container=$1
    if [ "$container" == "xconf-consul" ]; then
        docker run -d --name="$container"   \
            --hostname=consul0              \
            -p 8500:8500                    \
            -e CONSUL_BIND_INTERFACE=eth0   \
            $DOCKER_CONSUL_IMAGE
    elif [ "$container" == "xconf-etcd" ]; then
        docker run -d --name="$container"   \
            --hostname=member0              \
            -p 2379:2379                    \
            $DOCKER_ETCD_IMAGE              \
            /usr/local/bin/etcd -advertise-client-urls "http://$container:2379" -listen-client-urls http://0.0.0.0:2379
    else
        if [ ! -d "$SCRIPT_PATH/tls/certs" ]; then
            "$SCRIPT_PATH/tls/certs.sh"
        fi
        docker run -d --name="$container"       \
            --hostname=member0                  \
            -p 2389:2389                        \
            -v "$SCRIPT_PATH/tls/certs:/certs"  \
            $DOCKER_ETCD_IMAGE                  \
            /usr/local/bin/etcd -advertise-client-urls "https://$container:2389" -listen-client-urls https://0.0.0.0:2389 -cert-file /certs/etcd_server_cert.pem -key-file /certs/etcd_server_key.pem
    fi
}

# dockerPull pulls Consul / Etcd image.
dockerPull() {
    container=$1
    if [ "$container" == "xconf-consul" ]; then
       docker pull -q $DOCKER_CONSUL_IMAGE
    else
       docker pull -q $DOCKER_ETCD_IMAGE
    fi
}

# checkIsHealthy checks if consul/etcd containers are healthy after bringing them up.
checkIsHealthy() {
    container=$1
    retryNo=0
    maxRetries=5
    printf "\033[0;34m>>> Checking %s's health...\033[0m\n" "$container"
    while true ; do
        if [ "$container" == "xconf-consul" ]; then
            reply=$(curl -sS "http://localhost:8500/v1/health/node/consul0?filter=Status==passing" | grep '"Status": "passing"')
        elif [ "$container" == "xconf-etcd" ]; then
            reply=$(curl -sS http://localhost:2379/health | grep '"health":"true"')
        else 
            reply=$(curl -sS --cacert "$SCRIPT_PATH/tls/certs/ca_cert.pem" https://localhost:2389/health | grep '"health":"true"')
        fi
        if [ "$reply" != "" ]; then
            printf "\033[0;34m>>> %s is healthy\033[0m\n" "$container"
            break
        else
            echo ">>> $container is not healthy"  
        fi
        retryNo=$(( retryNo + 1 ))
        if [ $retryNo -eq $maxRetries ]; then
            printf '\033[0;31mFAIL\033[0m >>> %s is not healthy (%d retries)' "$container" "$retryNo"
            exit 1
        fi
        echo ">>> Sleeping ${retryNo}s and retrying..."
        sleep "$retryNo"
    done
}

setUpLocalDocker "xconf-consul"
setUpLocalDocker "xconf-etcd"
setUpLocalDocker "xconf-etcds"
