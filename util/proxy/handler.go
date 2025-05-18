package proxy

import (
	"fmt"
	"io"
	"net"
	"net/netip"
)

type Conn interface {
	io.Writer

	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

func getIP(addr net.Addr) (netip.AddrPort, Transport, error) {
	switch ipAddr := addr.(type) {
	case *net.UDPAddr:
		return ipAddr.AddrPort(), ProxyDgram, nil
	case *net.TCPAddr:
		return ipAddr.AddrPort(), ProxyStream, nil
	}
	return netip.AddrPort{}, 0, fmt.Errorf("unsupported address type: %T (%v)", addr, addr)
}

func WriteConn(c Conn) error {
	remoteAddr, proto, err := getIP(c.RemoteAddr())
	if err != nil {
		return err
	}
	localAddr, _, err := getIP(c.LocalAddr())
	if err != nil {
		return err
	}

	payload, err := MakePayload(proto, remoteAddr, localAddr)
	if err != nil {
		return err
	}
	_, err = c.Write(payload)
	return err
}
