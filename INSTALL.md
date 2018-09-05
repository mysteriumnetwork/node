---
## Mysterium VPN node (Any OS with Docker)

Most convenient way requiring least configuration is to run a node using [Docker](https://docs.docker.com/install/). 
All node versions are available through docker hub: 
 
https://hub.docker.com/r/mysteriumnetwork/mysterium-node/

### Fetching and running docker image
```bash
sudo docker run --cap-add NET_ADMIN --net host --name mysterium-node -d mysteriumnetwork/mysterium-node --agreed-terms-and-conditions
```

Note: to run server, you will have to accept terms & conditions by adding '--agreed-terms-and-conditions' command line option.
Note 2: it's mandatory to run docker container with --net host to correctly detect VPN service ip which needs to be published to clients, assuming that host on which node is running has external interface with public ip

### Debugging
```bash
sudo docker logs -f mysterium-node
```


## Mysterium VPN node (Debian && Ubuntu) - tested on Ubuntu 16.04
Note: you need to replace {version} with specific version number from [releases](https://github.com/mysteriumnetwork/node/releases/)
### Download
 * https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/mysterium-node_linux_amd64.deb

###  Add latest OpenVPN repository

```bash
apt-get update && apt-get install -y curl
curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list && rm -rf /var/cache/apt/* /var/lib/apt/lists/*
apt-get update
```

### Installation
```bash
wget https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/mysterium-node_linux_amd64.deb
sudo dpkg --install --force-depends mysterium-node_linux_amd64.deb
sudo apt-get install -y --fix-broken
```
### Running service
```bash
sudo service mysterium-node start
sudo service mysterium-node status
```
### Debugging service
```bash
sudo tail -f /var/log/mysterium-node/*
```
### Debugging standalone
```bash
sudo mysterium_server --data-dir=/var/lib/mysterium-node --config-dir=/etc/mysterium-node --runtime-dir=/tmp --identity=0x123456..
```


## Mysterium VPN client (Debian && Ubuntu) - tested on Ubuntu 16.04
### Download
 * https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/mysterium-client_linux_amd64.deb

### Add latest OpenVPN repository

```bash
apt-get update && apt-get install -y curl
curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list && rm -rf /var/cache/apt/* /var/lib/apt/lists/*
apt-get update
```

### Installation
```bash
wget https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/mysterium-client_linux_amd64.deb
sudo dpkg --install --force-depends mysterium-client_linux_amd64.deb
sudo apt-get install -y --fix-broken
```

In order for mysterium client to change system DNS servers to ones provided by VPN node
 it needs to modify /etc/resolv.conf
This change is performed by resolvconf utility. After resolvconf utility is installed, you need:
```bash
dpkg-reconfigure resolvconf
```
in order to be able to alter /etc/resolv.conf dinamically.

### Running service
```bash
sudo service mysterium-client start
sudo service mysterium-client status
```
### Debugging service
```bash
sudo tail -f /var/log/mysterium-client/*
```
### Debugging standalone
```bash
sudo mysterium_client --data-dir=/var/lib/mysterium-client --config-dir=/etc/mysterium-client --runtime-dir=/tmp
```


## Mysterium VPN node and client standalone binaries (.tar.gz)
### Client
#### Download
 * https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/mysterium_client_{OS}_{ARCH}.tar.gz
#### Extract
```bash
tar -xvzf mysterium_client_{OS}_{ARCH}.tar.gz
```
#### Running
```bash
cd mysterium_client_{OS}_{ARCH}
sudo ./mysterium_client
```

### Node
#### Download
 * https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/mysterium_server_{OS}_{ARCH}.tar.gz
#### Extract
```bash
tar -xvzf mysterium_server_{OS}_{ARCH}.tar.gz
```
#### Running
```bash
cd mysterium_server_{OS}_{ARCH}
sudo ./mysterium_server
```
Note: to run server, you will have to accept terms & conditions by adding '--agreed-terms-and-conditions' command line option.
