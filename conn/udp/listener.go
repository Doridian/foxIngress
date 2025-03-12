package udp

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
)

type Listener struct {
	addr    *net.UDPAddr
	udpConn *net.UDPConn
	proto   config.BackendProtocol

	listenCtx    context.Context
	listenCancel context.CancelFunc
	running      bool

	connLock sync.Mutex
	conns    map[string]*Conn
}

var _ conn.Listener = &Listener{}

func (l *Listener) Start() {
	// Yeah, this is a hack
	if l.listenCancel == nil {
		l.listenCtx, l.listenCancel = context.WithCancel(context.Background())
		l.running = true
		go l.reader()
	}

	<-l.listenCtx.Done()
}

func NewListener(addr string, proto config.BackendProtocol) (*Listener, error) {
	if proto != config.PROTO_QUIC {
		return nil, errors.New("UDP listener only supports QUIC")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		addr:    udpAddr,
		proto:   proto,
		udpConn: conn,
		conns:   make(map[string]*Conn),
	}
	return l, nil
}

func (l *Listener) removeConn(conn *Conn) {
	connKey := conn.remoteAddr.String()

	l.connLock.Lock()
	defer l.connLock.Unlock()

	delete(l.conns, connKey)
}

func (l *Listener) handlePacket(buf []byte, addr *net.UDPAddr) {
	connKey := addr.String()

	l.connLock.Lock()
	conn, ok := l.conns[connKey]
	if !ok || !conn.open {
		conn = &Conn{
			remoteAddr: addr,
			listener:   l,
			open:       true,
		}
		l.conns[connKey] = conn
	}
	l.connLock.Unlock()

	conn.handlePacket(buf)
}

func (l *Listener) reader() {
	buf := make([]byte, 65535)
	for l.running {
		n, addr, err := l.udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			_ = l.Close()
			return
		}

		go l.handlePacket(buf[:n], addr)
	}
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}

func (l *Listener) Close() error {
	l.connLock.Lock()
	defer l.connLock.Unlock()

	l.running = false
	l.listenCancel()
	for _, conn := range l.conns {
		_ = conn.Close()
	}

	return l.udpConn.Close()
}
