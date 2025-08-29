SOCKS5 and HTTP Proxy (Consumer Mode)

Overview
- In proxy mode the node can expose both an HTTP proxy and a SOCKS5 proxy simultaneously.
- HTTP proxy supports regular HTTP requests and HTTP CONNECT tunneling for arbitrary TCP.
- SOCKS5 proxy supports TCP CONNECT and UDP ASSOCIATE (no authentication).

How to Run
- Start node in proxy mode:
  - `myst --proxymode --proxymode.socks5` (the `--proxymode.socks5` switch enables SOCKS5 alongside HTTP)
- Bring up a connection and choose the base proxy port (for HTTP):
  - `myst connection up --proxy 10000 <provider_id>`

Ports
- HTTP proxy: listens on `127.0.0.1:<proxy_port>` (e.g., 10000).
- SOCKS5 proxy: listens on `127.0.0.1:<proxy_port+1>` (e.g., 10001). If the proxy port is 0 or unset, SOCKS5 defaults to 1080.

Client Configuration Examples
- HTTP proxy:
  - curl: `curl -x http://127.0.0.1:10000 https://ifconfig.me` 
- SOCKS5 proxy (no auth):
  - curl: `curl --socks5-hostname 127.0.0.1:10001 https://ifconfig.me`
  - apps supporting SOCKS5 UDP (e.g., some game launchers) can use `127.0.0.1:10001` with UDP enabled.

Notes
- SOCKS5 UDP replies embed a generic BND address (0.0.0.0:0); most clients ignore it. If you need actual source address in replies, open an issue.
- BIND command is not implemented (rarely used). CONNECT and UDP are supported.
- No authentication is implemented or required.

Functional Tests
- These are opt-in, require a running SOCKS5 proxy at `127.0.0.1:10001` (or override via env `SOCKS5_ADDR`). They are skipped automatically if the port isn’t listening.
- When skipped, running with `-v` shows an explicit skip message (e.g., `skipping functional test`) and the test run still passes.
- Start node and connect with a base proxy port (e.g., 10000) so SOCKS5 listens on 10001:
  - `./build/myst/myst service --agreed-terms-and-conditions` (add `--userspace` on macOS if needed)
  - `./build/myst/myst connection up --proxy 10000 <provider_id>`
- Run tests:
  - `go test ./testkit/socks5 -v`
- Env overrides:
  - `SOCKS5_ADDR` (default `127.0.0.1:10001`)
  - `DNS_HOST` for UDP/DNS test (default `1.1.1.1`)
  - `TCP_HOST` for TCP/CONNECT HTTP test (default `example.com`)
- To force tests to fail when SOCKS5 is not listening (useful in CI), set:
  - `SOCKS5_REQUIRED=1`
