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
	conns    map[connectionKey]*Conn
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
		conns:   make(map[connectionKey]*Conn),
	}
	return l, nil
}

func (l *Listener) removeConn(connObj *Conn) {
	connKey := makeConnKey(connObj.remoteAddr)

	l.connLock.Lock()
	defer l.connLock.Unlock()

	_, ok := l.conns[connKey]
	if !ok {
		return
	}

	l.removeConnInt(connObj, connKey)
}

func (l *Listener) removeConnInt(connObj *Conn, connKey connectionKey) {
	if connObj.backend != nil {
		conn.OpenConnections.WithLabelValues(l.proto.String(), l.IPProto(), l.addr.String(), connObj.backendMatch, connObj.backend.String()).Dec()
	}
	delete(l.conns, connKey)
}

func (l *Listener) handlePacket(buf []byte, addr *net.UDPAddr) {
	connKey := makeConnKey(addr)

	l.connLock.Lock()
	connObj, ok := l.conns[connKey]
	if !ok || !connObj.open {
		if ok {
			l.removeConnInt(connObj, connKey)
		}
		connObj = &Conn{
			remoteAddr: addr,
			listener:   l,
			open:       true,
		}
		l.conns[connKey] = connObj
		conn.RawConnectionsTotal.WithLabelValues(l.proto.String(), l.IPProto(), l.addr.String()).Inc()
	}
	l.connLock.Unlock()

	initial := connObj.handlePacket(buf)
	if initial {
		conn.ConnectionsTotal.WithLabelValues(l.proto.String(), l.IPProto(), l.addr.String(), connObj.backendMatch, connObj.backend.String()).Inc()
		conn.OpenConnections.WithLabelValues(l.proto.String(), l.IPProto(), l.addr.String(), connObj.backendMatch, connObj.backend.String()).Inc()
	}
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

func (l *Listener) IPProto() string {
	return "UDP"
}
