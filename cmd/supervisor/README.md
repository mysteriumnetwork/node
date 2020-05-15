# node-supervisor

Supervisor is a system background service that allows seamless installation/running of [Mysterium node](https://github.com/mysteriumnetwork/node) under elevated permissions.
Clients (e.g. desktop applications) can ask the supervisor to RUN or KILL the node instance via OS dependent mechanism.

Currently, only macOS is supported and it is using unix domain sockets for communication.

For usage see:

```
myst_supervisor -help
```

## Elevated command support table

| Command                           | OS           | Args | Output | Implemented | Notes |
| --------------------------------- | ------------ | ---- | ------ | ----------- | ----- |
| isWgSupported (kernel)            | linux        |      | ok     | -           | |
| NAT ipForward                     | linux, macOS |      | ok     | -           | | `/sbin/sysctl -w net.ipv4.ip_forward=1` |
| NAT/firewall rules                | ALL          | vpnNetwork <br> dnsIP <br> dnsPort <br> protectedNetworks <br> providerExternalIP <br> enableDNSRedirect | ok | - | |
| NAT/firewall allowIP              | linux        | IP   | ok     | -           | |
| NAT/firewall allowURL             | linux        | URL  | ok     | -           | |
| Create tun (wg userspace)         | ALL          | iface name | `windows.GUID / FD int` | - | FD to be used in `wg/tun.CreateTUNFromFile(*os.File, int) (Device, error)` see `tun_linux.go/tun_darwin.go` <br> windows: `CreateTUNWithRequestedGUID` |
| Destroy device (wg userspace)     | ALL          | iface name | ok | 
| assignIP                          | ALL          | iface name, subnet | ok |
| excludeRoute                      | ALL          | IP | ok |
| defaultRoute                      | ALL          | iface name | ok |
