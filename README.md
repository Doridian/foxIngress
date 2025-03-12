# foxIngress

Host-based HTTP(S) router. It can determine the target hostname (and route) connections using the following protocols.

For this, it does not need knowledge of any private keys, even for HTTPS and QUIC (which is the goal of this proxy)

- HTTP: **Host** header
- HTTPS: Client Hello **SNI**
- QUIC: Initial Packet **SNI**

By default it looks for a file called `config.yml` in the working directory, but this can be influenced with the `CONFIG_FILE` environment variable.

See [config.example.yml](config.example.yml) for an example config.

MIT licensed
