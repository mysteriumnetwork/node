FROM golang:1.9.2 AS builder

ARG PACKAGE_VERSION

# Install FPM
RUN apt-get update \
    && apt-get install -y ruby-dev build-essential \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* \
    && gem install ffi -v 1.9.18 \
    && gem install fpm -v 1.9.3

# Install Debber
RUN go get github.com/debber/debber-v0.3/cmd/debber

WORKDIR /go/src/github.com/mysterium/node

ENTRYPOINT ["/bin/bash"]
