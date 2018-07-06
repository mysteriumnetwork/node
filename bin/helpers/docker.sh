#!/bin/bash

# Uploads already created Docker image to Docker Hub
#
# Usage:
# > docker_release_image <version> [tag..]
#
# Uploads specific version:
# > docker_release_image mysteriumnetwork/mysterium-node alpine 0.0.1
#
# Uploads several versions
# > docker_release_image mysteriumnetwork/mysterium-node ubuntu ${VERSION}-ubuntu ubuntu
docker_release_image () {
    DOCKER_IMAGE=$1; shift;
    DOCKER_BUILD_TAG=$2; shift;

    while test $# -gt 0; do
        DOCKER_TAG=$1; shift;

        printf "Publishing '${DOCKER_TAG}' image..\n"
        docker tag ${DOCKER_IMAGE}:${DOCKER_BUILD_TAG} ${DOCKER_IMAGE}:${DOCKER_TAG}
        docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
    done
}
