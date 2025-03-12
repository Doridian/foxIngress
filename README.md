# sni-vhost-proxy

Host-based HTTP(S) router. It can determine the target hostname (and route) connections using the following protocols.

For this, it does not need knowledge of any private keys, even for HTTPS and QUIC (which is the goal of this proxy)

- HTTP: **Host** header
- HTTPS: TLS **SNI**
- QUIC: TLS **SNI**

MIT licensed
