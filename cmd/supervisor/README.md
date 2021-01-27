# node-supervisor

Supervisor is a system background service that allows seamless installation/running of [Mysterium node](https://github.com/mysteriumnetwork/node) under elevated permissions.
Clients (e.g. desktop applications) can ask the supervisor to RUN or KILL the node instance via OS dependent mechanism.

Currently, only macOS and Windows are supported and it is using unix domain sockets or named pipes for communication.

For usage see:

```
myst_supervisor -help
```

## Elevated command support table

| Command                           | OS           | Args | Output | Implemented | Notes |
| --------------------------------- | ------------ | ---- | ------ | ----------- | ----- |
| ping                              | macOS, Win   |      | ok     | ✅           | Ping supervisor |
| kill                              | macOS, Win   |      | ok     | ✅          | Kill myst process gracefully |
| bye                               | macOS   |      | ok     | ✅            | Kill supervisor |
| wg-up                             | macOS, Win   | -uid, -config    | ok     | ✅           | Setup WireGuard device with given configuration in JSON string encoded as base64 |
| wg-down                           | macOS, Win   | -iface     | ok     | ✅           | Destroy WireGuard device |
| wg-stats                          | macOS, Win   | -iface     | `{"bytes_send": 100, "bytes_received": 200, "last_handshake": "2020-06-02T13:42:55.786Z"}`     | ✅           | Get WireGuard device peer statistics |
| ta-set-port                       | macOS, Win   | port     | ok     | ✅           | Set tequilapi port for supervisor |


## Logs

On Windows logs could be found at `C:\ProgramData\MystSupervisor\myst_supervisor.log`

On macOS logs could be found at `/var/log/myst_supervisor.log`
