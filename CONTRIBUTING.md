# Contributing guide

## Development environment

* **Step 0.** Setup Golang dev environment

Mysterium Network Node is written in Golang, if you want to hack on it you will
need a functional go dev environment.  See official golang documentation for how
to https://golang.org/doc/install

* **Step 1.** Get project dependencies

Install openvpn

```bash
# OS X using brew
brew install openvpn

# Ubuntu
sudo apt-get install openvpn

```

* **Step 2.** Get the Mysterium-Node repository

One way to manage your repositories is to fork the `mysteriumnetwork/node`
repository to your GitHub account then clone your newly forked repository
(replace USER with your github username).

```bash
cd $GOPATH/src/github.com
mkdir mysteriumnetwork
cd mysteriumnetwork
git clone https://github.com/USER/node
```

This creates a remote called `origin`.  To keep your fork in sync with upstream
development add a remote called `upstream`
```bash
cd node
git remote add upstream https://github.com/mysteriumnetwork/node.git
```

For bonus points add a git alias and a shell alias to do the syncing.  Edit
`~/.gitconfig and add
```bash
[alias]
	pu = !"git fetch origin -v; git fetch upstream -v; git merge upstream/master"
```

Then define a shell alias
```bash
alias sync-repo='git pu; git push
```

Now any time you are on master branch (for any project you set up like this) you
can run `sync-repo` to sync your fork with the upstream repository.

* **Step 3.** Build

Run `make help` to get a list of build targets (you may like to run `make dep`
and `make dep-ensure` to install `dep` and to get dependencies).  Happy hacking!

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
dep ensure -add github.com/ccding/go-stun
dep ensure -add github.com/ccding/go-stun@^0.1.0
```

* Update package in project
```bash
vim Gopkg.toml
dep ensure -update
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
