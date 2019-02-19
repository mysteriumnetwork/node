## Mysterium VPN node (Any OS with Docker)

Most convenient way requiring least configuration is to run a node using [Docker](https://docs.docker.com/install/).
All node versions are available through docker hub:

https://hub.docker.com/r/mysteriumnetwork/myst/

### Fetching and running docker image
```bash
sudo docker run --cap-add NET_ADMIN --net host --name myst -d mysteriumnetwork/myst service --agreed-terms-and-conditions
```

>**Note:** to run server, you will have to accept terms & conditions by adding '--agreed-terms-and-conditions' command line option.
>
>**Note 2:** it's mandatory to run docker container with --net host to correctly detect VPN service ip which needs to be published to clients, assuming that host on which node is running has external interface with public ip

### Debugging
```bash
sudo docker logs -f myst
```

## Mysterium VPN node (Debian && Ubuntu) - tested on Ubuntu 16.04
>**Note:** you need to replace {version} with specific version number from [releases](https://github.com/mysteriumnetwork/node/releases/)

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
wget https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/myst_linux_amd64.deb
sudo dpkg --install --force-depends myst_linux_amd64.deb
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
sudo myst --data-dir=/var/lib/mysterium-node --config-dir=/etc/mysterium-node --runtime-dir=/tmp --identity=0x123456..
```

## Mysterium VPN node and client standalone binaries (.tar.gz)

#### Download
 * https://github.com/mysteriumnetwork/node/releases/download/{VERSION}/myst_{OS}_{ARCH}.tar.gz

#### Extract
```bash
tar -xvzf myst_{OS}_{ARCH}.tar.gz
```

#### Running
```bash
cd myst_{OS}_{ARCH}
sudo ./myst
```

>**Note:** to run server, you will have to add `service` subcommand and accept terms & conditions by adding '--agreed-terms-and-conditions' command line option.
