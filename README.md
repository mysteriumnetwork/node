---
# Decentralized VPN built on blockchain

VPN server and client for Mysterium Network 

- Homepage https://mysterium.network/
- [Whitepaper](https://mysterium.network/whitepaper.pdf)
- Release [latest](https://github.com/MysteriumNetwork/node/releases/latest)
- [Installation guides](./INSTALL.md)

## Mysterium VPN node (Any OS with Docker)
https://hub.docker.com/r/mysteriumnetwork/mysterium-node/
### Installation
```bash
sudo apt-get install docker.io
sudo docker run --cap-add NET_ADMIN --net host --publish "1194:1194" -e "NODE=123456" --name mysterium-node -d mysteriumnetwork/mysterium-node:latest
```
### Running
```bash
sudo docker start mysterium-node
sudo docker stop mysterium-node
```
### Debugging
```bash
sudo docker logs -f mysterium-node
```