.PHONY: server client

default: help

CMD_GODEP := $(shell which dep)
SERVER_DOCKERFILE = bin/server_docker/ubuntu/Dockerfile
SERVER_IMAGE_NAME = mysteriumnetwork/mysterium-node:latest
CLIENT_DOCKERFILE = bin/client_docker/ubuntu/Dockerfile
CLIENT_IMAGE_NAME = mysteriumnetwork/mysterium-client:latest

help:
	@echo "Select a sub command \n"
	@echo "godep:\t Install package manager 'godep'"
	@echo "dep:\t Get dependencies"
	@echo "server:\t Build Mysterium server"
	@echo "client:\t Build Mysterium client"
	@echo "server-image:\t Build Mysterium server Docker image"
	@echo "client-image:\t Build Mysterium client Docker image"
	@echo "help:\t Display this help"
	@echo "\nSee README.md for more."

godep:
	if [ "$(CMD_GODEP)" == "" ] ; then go get github.com/tools/godep ; fi

dep:
	dep ensure

server:
	./bin/server_build

client:
	./bin/client_build

server-image:
	docker build -t $(SERVER_IMAGE_NAME) -f $(SERVER_DOCKERFILE) .

client-image:
	docker build -t $(CLIENT_IMAGE_NAME) -f $(CLIENT_DOCKERFILE) .
