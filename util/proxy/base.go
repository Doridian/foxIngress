package proxy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
)

type Transport = byte

const (
	ProxyStream Transport = 0b00000001 // TCP
	ProxyDgram  Transport = 0b00000010 // UDP

	AFIPv4  = 0b00010000
	AFIPv6  = 0b00100000
	AFUnix  = 0b00110000
	AFUnset = 0b00000000

	// src address + src port + dst address + dst port
	addrLenIPv4 = (net.IPv4len + 2) * 2
	addrLenIPv6 = (net.IPv6len + 2) * 2

	protoVersion = 0b00100000 // 2
	protoCommand = 0b00000001 // PROXY
)

var header = [13]byte{
	0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A, // Signature
	protoVersion | protoCommand,
}

func MakePayload(proto Transport, srcAddr netip.AddrPort, dstAddr netip.AddrPort) ([]byte, error) {
	maxAddrLen := net.IPv4len
	if srcAddr.Addr().Is6() || dstAddr.Addr().Is6() {
		maxAddrLen = net.IPv6len
	}

	outBuf := bytes.Buffer{}
	outBuf.Write(header[:])

	switch maxAddrLen {
	case net.IPv4len:
		outBuf.WriteByte(AFIPv4 | proto)
		binary.Write(&outBuf, binary.BigEndian, uint16(addrLenIPv4))

		addr := srcAddr.Addr().As4()
		outBuf.Write(addr[:])
		addr = dstAddr.Addr().As4()
		outBuf.Write(addr[:])
	case net.IPv6len:
		outBuf.WriteByte(AFIPv6 | proto)
		binary.Write(&outBuf, binary.BigEndian, uint16(addrLenIPv6))

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
