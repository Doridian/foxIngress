package udp

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/util"
	"github.com/gaukas/clienthellod"
)

type Conn struct {
	remoteAddr *net.UDPAddr
	open       bool
	listener   *Listener

	readerTimeout *time.Timer

	beConn *net.UDPConn
}

var IdleTimeout = 60 * time.Second

func (c *Conn) handleInitial(buf []byte) {
	c.readerTimeout = time.NewTimer(IdleTimeout)
	go func() {
		<-c.readerTimeout.C
		_ = c.Close()
	}()

	qHello, err := clienthellod.ParseQUICCIP(buf)
	if err != nil {
		if config.Verbose {
			log.Printf("Error parsing QUIC IP: %v", err)
		}
		return
	}

	serverName := qHello.QCH.ServerName
	backend, err := config.GetBackend(serverName, config.PROTO_QUIC)
	if err != nil {
		log.Printf("Error finding backend: %v", err)
		_ = c.Close()
		return
	}

	if backend == nil {
		// This means we don't want to handle the connection
		_ = c.Close()
		return
	}

	useHost := backend.Host
	if backend.HostPassthrough {
		useHost = serverName
	}

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("[%s]:%d", useHost, backend.Port))
	if err != nil {
		log.Printf("Error resolving UDP address: %v", err)
		_ = c.Close()
		return
	}
	c.beConn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Printf("Error dialing UDP: %v", err)
		_ = c.Close()
		return
	}

	if backend.ProxyProtocol {
		payload, err := util.MakeProxyProtocolPayload(c.RemoteAddr().AddrPort(), c.LocalAddr().AddrPort())
		if err != nil {
			log.Printf("Error making proxy protocol payload: %v", err)
			_ = c.Close()
			return
		}
		_, err = c.Write(payload)
		if err != nil {
			log.Printf("Error writing proxy protocol payload: %v", err)
			_ = c.Close()
			return
		}
	}

	go c.beReader()
}

func (c *Conn) beReader() {
	buf := make([]byte, 65536)
	for c.open {
		n, _, err := c.beConn.ReadFromUDP(buf)
		if err != nil {
			if config.Verbose {
				log.Printf("Error reading from backend: %v", err)
			}
			_ = c.Close()
			return
		}

		c.readerTimeout.Reset(IdleTimeout)

		_, err = c.Write(buf[:n])
		if err != nil {
			if config.Verbose {
				log.Printf("Error writing to client: %v", err)
			}
			_ = c.Close()
			return
		}
	}
}

func (c *Conn) handlePacket(buf []byte) {
	if !c.open {
		return
	}

	if c.beConn == nil {
		c.handleInitial(buf)
		if c.beConn == nil {
			_ = c.Close()
			return
		}
	}

	_, err := c.beConn.Write(buf)
	if err != nil {
		if config.Verbose {
			log.Printf("Error writing to backend: %v", err)
		}
		_ = c.Close()
		return
	}
}

func (c *Conn) Close() error {
	c.open = false
	c.listener.removeConn(c)
	if c.beConn != nil {
		_ = c.beConn.Close()
	}
	return nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	if !c.open {
		return 0, net.ErrClosed
	}

	return c.listener.udpConn.WriteToUDP(b, c.remoteAddr)
}

func (c *Conn) LocalAddr() *net.UDPAddr {
	return c.listener.addr
}

func (c *Conn) RemoteAddr() *net.UDPAddr {
	return c.remoteAddr
}
