# Mysterium Node - decentralized VPN built on blockchain

[![Go Report Card](https://goreportcard.com/badge/github.com/mysteriumnetwork/node)](https://goreportcard.com/report/github.com/mysteriumnetwork/node)
[![pipeline status](https://gitlab.com/mysteriumnetwork/node/badges/master/pipeline.svg)](https://gitlab.com/mysteriumnetwork/node/pipelines)
[![pullreminders](https://pullreminders.com/badge.svg)](https://pullreminders.com?ref=badge)
[![codecov](https://codecov.io/gh/mysteriumnetwork/node/branch/master/graph/badge.svg)](https://codecov.io/gh/mysteriumnetwork/node) 
[![GoDoc](https://godoc.org/github.com/mysteriumnetwork/node?status.svg)](http://godoc.org/github.com/mysteriumnetwork/node)

Cross-platform software to run a node in Mysterium Network. It contains Mysterium server (node),
client API (tequila API) and client-cli (console client) for Mysterium Network.

Currently node supports OpenVPN as its underlying VPN transport.

## Getting Started

- Homepage https://mysterium.network
- [Whitepaper](https://mysterium.network/whitepaper.pdf)
- [Latest release](https://github.com/mysteriumnetwork/node/releases/latest)
- [Snapshot builds](https://github.com/mysteriumnetwork/node-builds/releases) - bleeding edge, use at your own risk
- [Documentation](http://docs.mysterium.network/en/latest/)
- [Installation guide](http://docs.mysterium.network/en/latest/user-guide/installation/)
- [Node wiki](https://github.com/mysteriumnetwork/node/wiki/)

## Installation options

### Debian / Ubuntu / Raspbian

Install latest stable release:
```bash
sudo -E bash -c "$(curl -s https://raw.githubusercontent.com/mysteriumnetwork/node/master/install.sh)" 
```

Or install latest snapshot (development build):
```bash
SNAPSHOT=true sudo -E bash -c "$(curl -s https://raw.githubusercontent.com/mysteriumnetwork/node/master/install.sh)" 
```

Service logs:
```bash
sudo journalctl -u mysterium-node.service
```

Service status:
```bash
sudo systemctl status mysterium-node.service
```

Installation script tested on these OSes so far: _Raspbian 10_, _Debian 9_, _Debian 10_, _Ubuntu 18.04_ and _Ubuntu 16.04_ .

### Docker

Our docker images can be found in [Docker hub](https://hub.docker.com/r/mysteriumnetwork/myst).

To run a node in a docker container you will need [docker](https://www.docker.com/). On Linux, to manage docker as a non-root user (execute commands without `sudo`), follow [postinstall guide](https://docs.docker.com/install/linux/linux-postinstall/).
You should be able to run a node on any OS that supports docker. We have tested it on these OSes so far:
- Debian 9
- Debian 10
- Ubuntu 16.04
- Ubuntu 18.04
- Centos 7

Run node:
```bash
docker run \
  --cap-add NET_ADMIN \
  --net host \
  --name myst -d \
  mysteriumnetwork/myst service --agreed-terms-and-conditions
```

Access service logs:
```bash
docker logs -f myst
```

### Further information

More installation options are described in the [installation guide](http://docs.mysterium.network/en/latest/user-guide/installation/).  
For possible issues while running a node refer to our [FAQ](https://github.com/mysteriumnetwork/node/wiki/Node-operation) section.

## Built With

* [go](https://golang.org/) - The Go Programming Language
* [travis](https://travis-ci.com/) - Travis continuous integration tool
* [docker](https://www.docker.com/what-docker) - Containerize applications
* [openvpn](https://openvpn.net) - Solid VPN solution
* [wireguard](https://www.wireguard.com/) - Extremely simple yet fast and modern VPN

## Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Contributors
* **Valdas Petrulis** - *Lead developer, go evangelist, node bootstrapper* - [Waldz](https://github.com/Waldz)
* **Tadas Valiukas** - *Senior developer, experienced bug maker* - [tadovas](https://github.com/tadovas)
* **Donatas Kučinskas** - *Senior developer, clean code savvy* - [donce](https://github.com/donce)
* **Antanas Masevičius** - *Network engineer / developer, net guru* - [zolia](https://github.com/zolia)
* **Paulius Mozuras** - *Software developer, snakes lover* - [interro](https://github.com/interro)
* **Ignas Bernotas** - *Senior developer, open source enthusiast* - [ignasbernotas](https://github.com/ignasbernotas)
* **Andrej Novikov** - *Senior developer, JS enthusiast, UX jazzman* - [shroomist](https://github.com/shroomist)
* **Dmitry Shihovtsev** - *Senior developer, devops ninja* - [soffokl](https://github.com/soffokl)
* **Viktoras Kuznecovas** - *Senior developer, supersonic typing specialist* [vkuznecovas](https://github.com/vkuznecovas)
* **Tadas Krivickas** - *Senior developer, CI boss, refactoring fairy* [tadaskay](https://github.com/tadaskay)
* **Jaro Šatkevič** - *Senior developer, micro-payments researcher, crypto maniac* [chompomonim](https://github.com/chompomonim)
* **Andzej Maciusovič** - *Senior developer, disciplined world changer* [anjmao](https://github.com/anjmao)

See also full list of [contributors](https://github.com/mysteriumnetwork/node/contributors) who participated in this project.

## License

This project is licensed under the terms of the GNU General Public License v3.0 (see [details](./LICENSE)).
