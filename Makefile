.PHONY: server client

default: help

CMD_DEP := $(shell which dep)
SERVER_DOCKERFILE = bin/server_docker/ubuntu/Dockerfile
SERVER_IMAGE_NAME = mysteriumnetwork/mysterium-node:latest
CLIENT_DOCKERFILE = bin/client_docker/ubuntu/Dockerfile
CLIENT_IMAGE_NAME = mysteriumnetwork/mysterium-client:latest

help:
	@echo "Select a sub command \n"
	@echo "dep:\t Install package manager 'dep'"
	@echo "deps:\t Get dependencies"
	@echo "server:\t Build Mysterium server"
	@echo "client:\t Build Mysterium client"
	@echo "server-image:\t Build Mysterium server Docker image"
	@echo "client-image:\t Build Mysterium client Docker image"
	@echo "help:\t Display this help"
	@echo "\nSee README.md for more."

dep:
	if [ "$(CMD_DEP)" == "" ] ; then curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh ; fi

deps:
	dep ensure

server:
	./bin/server_build

client:
	./bin/client_build

server-image:
	docker build -t $(SERVER_IMAGE_NAME) -f $(SERVER_DOCKERFILE) .

client-image:
	docker build -t $(CLIENT_IMAGE_NAME) -f $(CLIENT_DOCKERFILE) .
