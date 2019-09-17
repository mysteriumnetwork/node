.PHONY: server client

default: help

help:
	@echo "Select a sub command \n"
	@echo "build:\t Build myst"
	@echo "build-image:\t Build myst Docker image"
	@echo "test:\t Run unit tests"
	@echo "help:\t Display this help"
	@echo "\nSee README.md for more."

build: FORCE
	./bin/build

build-image:
	./bin/package_docker

test:
	./bin/test

FORCE: ;

