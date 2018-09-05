FROM golang:1.9.2 AS builder

# Install FPM
RUN apt-get update \
    && apt-get install -y ruby-dev build-essential \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* \
    && gem install ffi -v 1.9.18 \
    && gem install fpm -v 1.9.3

# Install development dependencies
RUN go get \
    github.com/debber/debber-v0.3/cmd/debber \
    github.com/golang/lint/golint \
    golang.org/x/tools/cmd/goimports \
    github.com/go-swagger/go-swagger/cmd/swagger

WORKDIR /go/src/github.com/mysteriumnetwork/node

ENTRYPOINT ["/bin/bash"]
