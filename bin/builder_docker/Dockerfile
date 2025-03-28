FROM golang:1.24 AS builder

# Install FPM
RUN apt-get update \
    && apt-get install -y ruby-dev build-essential \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* \
    && gem install ffi -v 1.10.0 \
    && gem install fpm -v 1.12.0

# Install development dependencies
RUN go install \
    github.com/debber/debber-v0.3/cmd/debber@latest \
    golang.org/x/lint/golint@latest \
    golang.org/x/tools/cmd/goimports@latest \
    github.com/go-swagger/go-swagger/cmd/swagger@latest

WORKDIR /go/src/github.com/mysteriumnetwork/node

ENTRYPOINT ["/bin/bash", "-c"]
