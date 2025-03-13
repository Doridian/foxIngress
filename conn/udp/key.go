package udp

import "net"

type connectionKey string

func makeConnKey(addr *net.UDPAddr) connectionKey {
	return connectionKey(addr.String())
}
