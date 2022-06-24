#!/bin/sh

#
# This script is used in Github Workflow to clean up Docker started containers.
#
# It is needed as a warning would appear in Github default "Stop containers" task, like
# "Error response from daemon: error while removing network: network github_network_a75750cc24824882af4d4b12d497d6e4 id d76f9ebb0f378f8c039925e82731fc0ace6d390fa7c25f80219594b9a9d3dd4f has active endpoints
# Warning: Docker network rm failed with exit code 1"
#

docker stop integration-consul
docker rm integration-consul
docker stop integration-etcd
docker rm integration-etcd
