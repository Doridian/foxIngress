package udpconn

import (
	"log"
	"net"
	"sync"
	"time"
)

type Conn struct {
	remoteAddr *net.UDPAddr
	open       bool
	listener   *Listener

	readerTimeout *time.Timer

	packets    [][]byte
	packetWait sync.Cond
	readLock   sync.Mutex
}

var _ net.Conn = &Conn{}

var IdleTimeout = 60 * time.Second

func (c *Conn) handlePacket(buf []byte) {
	if c.readerTimeout == nil {
		c.readerTimeout = time.NewTimer(IdleTimeout)
		go func() {
			<-c.readerTimeout.C
			c.Close()
		}()
	} else {
		c.readerTimeout.Reset(IdleTimeout)
	}

	c.buf = append(c.buf, buf...)
	c.packetWait.Broadcast()
}

func (c *Conn) Close() error {
	c.open = false
	c.packetWait.Broadcast()
	c.listener.removeConn(c)
	log.Printf("Conn closed: %v -> %v", c.LocalAddr(), c.RemoteAddr())
	return nil
}

func (c *Conn) Read(b []byte) (n int, err error) {
	c.readLock.Lock()
	defer c.readLock.Unlock()

	for len(c.buf) == 0 && c.open {
		c.packetWait.Wait()
	}

	if !c.open {
		return 0, net.ErrClosed
	}

	n = copy(b, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	if !c.open {
		return 0, net.ErrClosed
	}

	return c.listener.listener.WriteToUDP(b, c.remoteAddr)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.listener.addr
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}
