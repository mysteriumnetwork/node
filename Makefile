.PHONY: server client

default: help

CMD_DEP := $(shell which dep)

help:
	@echo "Select a sub command \n"
	@echo "dep:\t Install package manager 'dep'"
	@echo "dep-ensure:\t Get dependencies"
	@echo "build:\t Build myst"
	@echo "build-image:\t Build myst Docker image"
	@echo "help:\t Display this help"
	@echo "\nSee README.md for more."

dep:
	if [ "$(CMD_DEP)" = "" ] ; then curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh ; fi

dep-ensure:
	dep ensure

build: FORCE
	./bin/build

build-image:
	./bin/package_docker

FORCE: ;

