# Contributing guide


## Development environment

* **Step 1.** Get Golang
```bash
brew install go
brew install glide

export GOPATH=~/workspace/go
git clone git@github.com:MysteriumNetwork/node.git $GOPATH/src/github.com/mysterium/node
cd $GOPATH/src/github.com/mysterium/node
```

* **Step 2.** Compile code
```bash
glide install
```

* **Step 3.** Prepare configuration

Enter `MYSTERIUM_API_URL` address of running [api](https://github.com/MysteriumNetwork/api) instance<br/>
Enter `NATS_SERVER_IP` address of running [communication broker](https://github.com/nats-io/gnatsd) instance

```bash
cp .env_example .env
vim .env
go build github.com/mysterium/node/...
```

## Running

```bash
# Start communication broker
docker-compose up broker

# Start node
bin/server_build
bin/server_run

# Client connects to node
bin/client_build
bin/client_run
```

## Running client in interactive cli

```bash
# Start client with --cli
bin/client_run_cli

# Show commands
» help
[INFO] Mysterium CLI tequilapi commands:
  connect
  identities
  ├── new
  ├── list
  status
  proposals
  ip
  disconnect
  help
  quit
  stop
  unlock

# Create a customer identity
» identities new

# Unlock a customer identity
» unlock <identity>

# Show provider identities
» proposals

# Connect to a server
» connect <consumer-identity> <provider-identity>
```

## Generate Tequila API documentation from client source code

* **Step 1.** Install go-swagger
```bash
brew tap go-swagger/go-swagger
brew install go-swagger
```

* **Step 2.** Generate specification and serve serve it locally:
```bash
bin/swagger_serve_doc
```

## Dependency management

* Install project's frozen packages
```bash
glide install
glide install --force
```

* Add new package to project
```bash
glide get --quick github.com/ccding/go-stun
```

* Update package in project
```bash
vim glide.yaml
glide update
```


## Debian packaging

* **Step 1.** Get FPM tool
See http://fpm.readthedocs.io/en/latest/installing.html

```bash
brew install gnu-tar
gem install --no-ri --no-rdoc fpm
```

* **Step 2.** Get Debber tool
See https://github.com/debber/debber-v0.3

```bash
go get github.com/debber/debber-v0.3/cmd/...
```

* **Step 3.** Build .deb package
```bash
bin/server_package_debian 0.0.6 amd64
bin/client_package_debian 0.0.6 amd64
```

## Creating pull request

To contribute a code, fist you must create a pull request (PR). If your changes will be accepted
this PR will be merged into main branch.

Before creating PR be sure to: 

* **Step 1.** Ensure that **your** code quality is passing

```bash
bin/check
```

* **Step 2.** Ensure that all unit tests pass

```bash
bin/test
```

* **Step 3.** Ensure that all end-to-end tests pass

```bash
bin/test_e2e
```

After you forked a project, modified sources and run tests, you can create a pull request using this procedure:
 
 https://help.github.com/articles/creating-a-pull-request-from-a-fork/
