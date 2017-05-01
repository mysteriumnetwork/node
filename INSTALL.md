---

## Mysterium VPN node (Debian && Ubuntu)
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_amd64.deb
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_i386.deb
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_armhf.deb
### Installation
```bash
wget https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-node_linux_amd64.deb
sudo dpkg --install mysterium-node_linux_amd64.deb
sudo apt-get install --fix-broken
```
### Running
```bash
service mysterium-node start
service mysterium-node status
```


## Mysterium VPN client (Debian && Ubuntu)
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_linux_amd64.deb
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_linux_i386.deb
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_linux_armhf.deb

### Installation
```bash
curl https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium-client_{VERSION}_linux_amd64.deb
sudo dpkg --install mysterium-client_{VERSION}_linux_amd64.deb
sudo apt-get install --fix-broken
```
### Running
```bash
service mysterium-client start
service mysterium-client status
```


## Mysterium VPN node (standalone Linux binaries)
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_server_linux_amd64
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_server_linux_386
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_server_linux_arm
### Running
```bash
mysterium_server --help
sudo mysterium_server --config-dir=/etc/mysterium-node --node=12345
```


## Mysterium VPN client (standalone Linux binaries)
### Download
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_linux_amd64
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_linux_386
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_linux_arm
### Running
```bash
mysterium_client --help
sudo mysterium_client --node=12345
```


## Mysterium VPN client (standalone Apple Mac/OSX/Darwin binaries)
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_darwin_amd64
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_darwin_386


### Mysterium VPN client (standalone Windows binaries)
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_windows_amd64.exe
 * https://github.com/MysteriumNetwork/node/releases/download/{VERSION}/mysterium_client_windows_386.exe


## Build from source code
 * https://github.com/MysteriumNetwork/node/archive/{VERSION}.tar.gz
 * https://github.com/MysteriumNetwork/node/archive/{VERSION}.zip