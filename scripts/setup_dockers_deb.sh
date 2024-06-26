#!/bin/sh

#
# This script is used in Github Workflow to run integration tests.
#
# It's Debian based, as it's the recommended system by Github.
# It's used instead of Github Service Containers because Etcd image needs to be run with CMD
# and Github Service Containers syntax does not support that.
# I got inspired from here: https://stackoverflow.com/questions/60849745/how-can-i-run-a-command-in-github-action-service-containers
#


# debug prints the passed message in blue color.
# Example: debug "Some debug message"
debug() {
    printf "\033[0;34m>>> %s\033[0m\n" "$1"
}

# checkIsHealthy checks if consul/etcd containers are healthy after bringing them up.
checkIsHealthy() {
    container=$1
    retryNo=0
    maxRetries=5
    debug "Checking ${container}'s health..."
    while true ; do
        if [ "$container" = "xconf-consul" ]; then
            reply=$(curl -sS "http://${container}:8500/v1/health/node/consul0?filter=Status==passing" | grep '"Status": "passing"')
        elif [ "$container" = "xconf-etcd" ]; then
            reply=$(curl -sS "http://${container}:2379/health" | grep '"health":"true"')
        else 
            reply=$(curl -sS --cacert "${GITHUB_WORKSPACE}/scripts/tls/certs/ca_cert.pem" "https://${container}:2389/health" | grep '"health":"true"')
        fi
        if [ "$reply" != "" ]; then
            debug "$container is healthy"
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

debug "Install deps"
# we need docker, curl and jq
# docker install: https://docs.docker.com/engine/install/debian/
apt-get update
apt-get install -y --no-install-recommends  \
    ca-certificates                         \
    curl                                    \
    gnupg                                   \
    lsb-release                             \
    jq

mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update
apt-get install -y --no-install-recommends  \
    docker-ce                               \
    docker-ce-cli                           \
    containerd.io                           \
    docker-compose-plugin

# find the network
hostName=$(hostname)
network=$(docker inspect --format '{{json .NetworkSettings.Networks}}' "${hostName}" | jq -r 'keys[0]')
echo ">>> network = ${network}"

debug "Run Consul Docker Image"
DOCKER_CONSUL_IMAGE_VER=hashicorp/consul:1.18.1
docker pull -q $DOCKER_CONSUL_IMAGE_VER
docker run -d                       \
    --name=xconf-consul             \
    --hostname=consul0              \
    --network "${network}"          \
    -e CONSUL_BIND_INTERFACE=eth0   \
    $DOCKER_CONSUL_IMAGE_VER

debug "Run Etcd Docker Image"
ETCD_VER=v3.5.13
DOCKER_ETCD_IMAGE_VER=quay.io/coreos/etcd:${ETCD_VER}
docker pull -q $DOCKER_ETCD_IMAGE_VER
docker run -d               \
    --name=xconf-etcd       \
    --hostname=member0      \
    --network "${network}"  \
    $DOCKER_ETCD_IMAGE_VER  \
    /usr/local/bin/etcd -advertise-client-urls http://xconf-etcd:2379 -listen-client-urls http://0.0.0.0:2379

debug "Run Etcd (with TLS) Docker Image"
if [ ! -d "${GITHUB_WORKSPACE}/scripts/tls/certs" ]; then
    "${GITHUB_WORKSPACE}/scripts/tls/certs.sh"
fi
docker build -q                                                 \
    -f "${GITHUB_WORKSPACE}/scripts/Dockerfile.etcdtls.github"  \
    -t xconf_etcds_image                                        \
    --build-arg ETCD_VER=${ETCD_VER}                            \
    "${GITHUB_WORKSPACE}"
docker run -d                       \
    --name=xconf-etcds              \
    -p 2389:2389                    \
    --hostname=member0              \
    --network "${network}"          \
    xconf_etcds_image

# Check healthiness
checkIsHealthy "xconf-consul"
checkIsHealthy "xconf-etcd"
checkIsHealthy "xconf-etcds"
