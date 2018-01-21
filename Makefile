.PHONY: server client

default: help

CMD_GLIDE := $(shell which glide)
BUILD_OUTPUT = build

help:
	@echo "Select a sub command \n"
	@echo "glide:\t Install package manager 'glide'"
	@echo "dep:\t Get dependencies"
	@echo "server:\t Build Mysterium server"
	@echo "client:\t Build Mysterium client"
	@echo "help:\t Display this help"
	@echo "\nSee README.md for more."

glide:
	if [ "$(CMD_GLIDE)" == "" ] ; then curl https://glide.sh/get | sh ; fi

dep:
	glide install

server:
	go build -o $(BUILD_OUTPUT)/server cmd/mysterium_server/mysterium_server.go

client:
	go build -o $(BUILD_OUTPUT)/client cmd/mysterium_client/mysterium_client.go
