#!/bin/sh

#
# This script is used in Github Workflow to clean up Docker started containers.
# It can be used also locally.
#
# It is needed as a warning would appear in Github default "Stop containers" task, like
# "Error response from daemon: error while removing network: network github_network_a75750cc24824882af4d4b12d497d6e4 id d76f9ebb0f378f8c039925e82731fc0ace6d390fa7c25f80219594b9a9d3dd4f has active endpoints
# Warning: Docker network rm failed with exit code 1"
#

# removeContainersByRegex stops and deletes container(s) that match(es) given regular expression.
# Example: removeContainersByRegex "my-container"
removeContainersByRegex() {
    containersRegex=$1
    existing=$(docker container ls -a | awk '{print $NF}' | grep -E "$containersRegex")
    if [ "$existing" != "" ]; then
        printf "\033[0;34m>>> Removing containers...\033[0m\n"
        docker ps | awk '{print $NF}' | grep -E "$containersRegex" | xargs docker stop > /dev/null
        docker container ls -a | awk '{print $NF}' | grep -E "$containersRegex" | xargs docker rm
    fi
}

removeContainersByRegex "^xconf-"
