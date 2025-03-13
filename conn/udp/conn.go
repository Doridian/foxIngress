package udp

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
	"github.com/Doridian/foxIngress/util"
	"github.com/gaukas/clienthellod"
)

type Conn struct {
	remoteAddr *net.UDPAddr
	open       bool
	listener   *Listener

	readerTimeout *time.Timer

	backend      *config.BackendInfo
	backendMatch string
	beConn       *net.UDPConn

	inPackets chan []byte
	ipBuff    []byte
}

var IdleTimeout = 60 * time.Second

const MaxPreBuff = 65536

func (c *Conn) handleInitial(pkt []byte) {
	if len(c.ipBuff) >= MaxPreBuff {
		if config.Verbose {
			log.Printf("QUIC IP buffer overflow, closing connection")
		}
		_ = c.Close()
		return
	}
	c.ipBuff = append(c.ipBuff, pkt...)

	qHello, err := clienthellod.ParseQUICCIP(c.ipBuff)
	if err != nil {
		if config.Verbose {
			log.Printf("Error parsing QUIC IP: %v", err)
		}
		return
	}

	serverName := qHello.QCH.ServerName
	c.backend, c.backendMatch, err = config.GetBackend(serverName, config.PROTO_QUIC)
	if err != nil {
		log.Printf("Error finding backend: %v", err)
		_ = c.Close()
		return
	}

	if c.backend == nil {
		// This means we don't want to handle the connection
		_ = c.Close()
		return
	}

	useHost := c.backend.Host
	if c.backend.HostPassthrough {
		useHost = serverName
	}

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("[%s]:%d", useHost, c.backend.Port))
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

	if c.backend.ProxyProtocol {
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

func (c *Conn) chReader() {
	defer c.Close()
	defer close(c.inPackets)

	for c.open {
		buf := <-c.inPackets

		if c.beConn == nil {
			c.handleInitial(buf)
			if c.beConn == nil {
				continue
			}
			buf = c.ipBuff
			c.ipBuff = nil

			conn.ConnectionsTotal.WithLabelValues(c.listener.proto.String(), c.listener.IPProto(), c.listener.addr.String(), c.backendMatch, c.backend.String()).Inc()
			conn.OpenConnections.WithLabelValues(c.listener.proto.String(), c.listener.IPProto(), c.listener.addr.String(), c.backendMatch, c.backend.String()).Inc()
			defer conn.OpenConnections.WithLabelValues(c.listener.proto.String(), c.listener.IPProto(), c.listener.addr.String(), c.backendMatch, c.backend.String()).Dec()
		}

		_, err := c.beConn.Write(buf)
		if err != nil {
			if config.Verbose {
				log.Printf("Error writing to backend: %v", err)
			}
			return
		}
	}
}

func (c *Conn) init() {
	c.ipBuff = make([]byte, 0)
	c.inPackets = make(chan []byte, 100)

	c.readerTimeout = time.NewTimer(IdleTimeout)
	go func() {
		<-c.readerTimeout.C
		_ = c.Close()
	}()

	c.open = true

	go c.chReader()
}

func (c *Conn) handlePacket(buf []byte) {
	if !c.open {
		return
	}

	c.readerTimeout.Reset(IdleTimeout)

	c.inPackets <- buf
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
