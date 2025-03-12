package util

import "net"

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
