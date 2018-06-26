---
## Mysterium VPN node (Any OS with Docker)

Most convenient way requiring least configuration is to run a node using docker. 
All node versions are available through docker hub: 
 
https://hub.docker.com/r/mysteriumnetwork/mysterium-node/
### Installation

Read on how to setup docker repository for Debian [here](https://docs.docker.com/install/linux/docker-ce/debian/) 
and for Ubuntu [here](https://docs.docker.com/install/linux/docker-ee/ubuntu/).

##### Debian / Ubuntu
```bash
sudo apt-get install docker-ce
```

##### CentOS / RedHat

Read on how to setup docker repository for CentOS [here](https://docs.docker.com/install/linux/docker-ce/centos/)
and for RedHat [here](https://docs.docker.com/install/linux/docker-ee/rhel/).

```bash
sudo yum install docker-ce
```

##### Fetching and running docker image
```bash
sudo docker run --cap-add NET_ADMIN --net host --publish "1194:1194" --name mysterium-node -d mysteriumnetwork/mysterium-node:{VERSION}
```
You can skip `{VERSION}` to fetch latest image. 

### Running
```bash
sudo docker start mysterium-node
sudo docker stop mysterium-node
```
### Debugging
```bash
sudo docker logs -f mysterium-node
```


## Mysterium VPN node (Debian && Ubuntu) - tested on Ubuntu 16.04
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_amd64.deb
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_armhf.deb

### Install latest OpenVPN and dependencies

```bash
apt-get update && apt-get install -y curl
curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list && rm -rf /var/cache/apt/* /var/lib/apt/lists/*
apt-get update
apt-get install openvpn ca-certificates iproute2 sudo
```

### Installation
```bash
wget https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_amd64.deb
sudo dpkg --install --force-depends mysterium-node_linux_amd64.deb
sudo apt-get install --fix-broken
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
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_linux_amd64.deb
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_linux_armhf.deb

### Install latest OpenVPN and dependencies

```bash
apt-get update && apt-get install -y curl
curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list && rm -rf /var/cache/apt/* /var/lib/apt/lists/*
apt-get update
apt-get install openvpn resolvconf ca-certificates iproute2 sudo
```

### Installation
```bash
wget https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_linux_amd64.deb
sudo dpkg --install --force-depends mysterium-client_linux_amd64.deb
sudo apt-get install --fix-broken
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


## Mysterium VPN node (standalone Linux binaries)
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_server_linux_amd64
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_server_linux_armhf
### Running
```bash
mysterium_server --help
sudo mysterium_server --config-dir=/etc/mysterium-node --identity=0x123456..
```


## Mysterium VPN client (standalone Linux binaries)
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_linux_amd64
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_linux_armhf
### Running
```bash
mysterium_client --help
sudo mysterium_client && mysterium_client --cli
```


## Mysterium VPN client (standalone Apple Mac/OSX/Darwin binaries)
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_osx_amd64


## Build from source code
 * https://github.com/MysteriumNetwork/node/archive/{VERSION}.tar.gz
 * https://github.com/MysteriumNetwork/node/archive/{VERSION}.zip
