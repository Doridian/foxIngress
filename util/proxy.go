package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
)

// Last bit here indicates version 2, PROXIED
var ProxyProtocolHeader = [13]byte{0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A, 0b00100001}

// These always indicate STREAM (TCP)
const ProxyAFIPv4 = 0b00010001
const ProxyAFIPv6 = 0b00100001

// src address + src port + dst address + dst port
const AddrLenIPv4 = (net.IPv4len + 2) * 2
const AddrLenIPv6 = (net.IPv6len + 2) * 2

type BackendInfo struct {
	Host            string
	ProxyProtocol   bool
	Port            int
	HostPassthrough bool
}

func MakeProxyProtocolPayload(srcAddr netip.AddrPort, dstAddr netip.AddrPort) ([]byte, error) {
	maxAddrLen := net.IPv4len
	if srcAddr.Addr().Is6() || dstAddr.Addr().Is6() {
		maxAddrLen = net.IPv6len
	}

	outBuf := bytes.Buffer{}
	outBuf.Write(ProxyProtocolHeader[:])

	switch maxAddrLen {
	case net.IPv4len:
		outBuf.WriteByte(ProxyAFIPv4)
		binary.Write(&outBuf, binary.BigEndian, uint16(AddrLenIPv4))

		addr := srcAddr.Addr().As4()
		outBuf.Write(addr[:])
		addr = dstAddr.Addr().As4()
		outBuf.Write(addr[:])
	case net.IPv6len:
		outBuf.WriteByte(ProxyAFIPv6)
		binary.Write(&outBuf, binary.BigEndian, uint16(AddrLenIPv6))

		addr := srcAddr.Addr().As16()
		outBuf.Write(addr[:])
		addr = dstAddr.Addr().As16()
		outBuf.Write(addr[:])
	default:
		return nil, fmt.Errorf("unknown address family len %d", maxAddrLen)
	}

	binary.Write(&outBuf, binary.BigEndian, srcAddr.Port())
	binary.Write(&outBuf, binary.BigEndian, dstAddr.Port())

	return outBuf.Bytes(), nil
}
