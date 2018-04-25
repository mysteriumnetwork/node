# Contributing guide


Development environment
------------
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
go build github.com/mysterium/node
```

* **Step 3.** Prepare configuration

Enter `MYSTERIUM_API_URL` value of running [api](https://github.com/MysteriumNetwork/api) instance

```bash
cp .env_example .env
vim .env
```

Running
------------
``` bash
# Start communication broker
docker-compose up broker

# Start node
bin/server_build
bin/server_run

# Client connects to node
bin/client_build
bin/client_run
```

Running client in interative cli
------------
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

Dependency management
------------
* Install project's frozen packages
```bash
glide install
glide install --force
```

* Add new package to project
```bash
glide get github.com/ccding/go-stun
```

* Update package in project
```bash
vim glide.yaml
glide update
```


Debian packaging
------------
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
