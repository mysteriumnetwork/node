# Contributing guide


## Development environment

* **Step 1.** Get project dependencies
```bash
brew install go
brew install godep
brew install openvpn

export GOPATH=~/workspace/go
git clone git@github.com:MysteriumNetwork/node.git $GOPATH/src/github.com/mysterium/node
cd $GOPATH/src/github.com/mysterium/node
```

* **Step 2.** Fetch dependencies
```bash
dep ensure
```

* **Step 3.** Start localnet infrastructure
```bash
bin/localnet/setup.sh
```

* **Step 4. (optional)** Tear down localnet infrastructure when it's not needed
```bash
bin/localnet/teardown.sh
```

## Running

```bash

# Start node
bin/server_build && bin/server_run

# Client connects to node
bin/client_build && bin/client_run
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
dep ensure
```

* Add new package to project
```bash
dep add github.com/ccding/go-stun
```

* Update package in project
```bash
vim Gopkg.toml
dep ensure
```

## Creating pull request

To contribute a code, first you must create a pull request (PR). If your changes will be accepted
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
