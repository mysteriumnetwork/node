#!/bin/bash

# Uploads already created Docker image to Docker Hub
#
# Usage:
# > docker_release_image <image-local> <image-release>
#
# Uploads specific tag:
# > docker_release_image mysterium-node:alpine mysteriumnetwork/mysterium-node:0.0.1
docker_release_image () {
    IMAGE_LOCAL=$1;
    IMAGE_RELEASE=$2;

    printf "Publishing '${IMAGE_RELEASE}' image..\n"
    docker tag ${IMAGE_LOCAL} ${IMAGE_RELEASE}
    docker push ${IMAGE_RELEASE}
}
