.PHONY: server client

default: help

help:
	@echo "Select a sub command \n"
	@echo "test:\t Run unit tests"
	@echo "build:\t Build myst"
	@echo "build-image:\t Build myst Docker image"
	@echo "help:\t Display this help"
	@echo "\nSee README.md for more."

test:
	./bin/test

build: FORCE
	./bin/build

build-image:
	./bin/package_docker

FORCE: ;

