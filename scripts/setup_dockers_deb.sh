#!/bin/sh -eux

#
# This script is used in Github Workflow to run integration tests.
#
# It's Debian based, as it's the recommended system by Github.
# It's used instead of Github Service Containers because Etcd image needs to be run with CMD
# and Github Service Containers syntax does not support that.
# I got inspired from here: https://stackoverflow.com/questions/60849745/how-can-i-run-a-command-in-github-action-service-containers
#

echo ">>> Install deps"
# we need docker, curl and jq
# docker install: https://docs.docker.com/engine/install/debian/
apt-get update
apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    jq

mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update
apt-get install -y --no-install-recommends \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-compose-plugin

# find the network
network=$(docker inspect --format '{{json .NetworkSettings.Networks}}' `hostname` | jq -r 'keys[0]')
echo "network = ${network}"

echo ">>> Run Consul Docker Image"
DOCKER_CONSUL_IMAGE_VER=consul:1.13.1
docker pull -q $DOCKER_CONSUL_IMAGE_VER
docker run -d \
    --name=integration-consul \
    -p 8500:8500 \
    --network "${network}" \
    -e CONSUL_BIND_INTERFACE=eth0 \
    $DOCKER_CONSUL_IMAGE_VER

echo ">>> Run Etcd Docker Image"
DOCKER_ETCD_IMAGE_VER=quay.io/coreos/etcd:v3.5.5
docker pull -q $DOCKER_ETCD_IMAGE_VER
docker run -d \
    --name=integration-etcd \
    -p 2379:2379 \
    --network "${network}" \
    $DOCKER_ETCD_IMAGE_VER \
    /usr/local/bin/etcd -advertise-client-urls http://integration-etcd:2379 -listen-client-urls http://0.0.0.0:2379

echo ">>> Show Running Docker Containers"
sleep 15
docker ps
