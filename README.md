simple-proxy
============

A simple routing proxy in Go.  Accepts incoming connections on ports 80 and 443.

* Connections on port 80 are assumed to be HTTP.  A hostname is extracted from each using
the HTTP "Host" header.
* Connections on port 443 are assumed to be TLS.  A hostname is extracted from the
server name indication in the ClientHello bytes.  Currently non-TLS SSL connections
and TLS connections without SNIs are dropped messily.

Once a hostname has been extracted from the incoming connection, the proxy looks up
a set of backends on a consul server, which is assumed to be running on 127.0.0.1:8500.
The key for the set is `protocall/domain/ i.e. https/test.example.com for https://test.example.com`

MIT licensed
