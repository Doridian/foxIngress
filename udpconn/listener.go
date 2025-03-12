package udpconn

import (
	"context"
	"errors"
	"net"
	"sync"
)

type Listener struct {
	addr     *net.UDPAddr
	listener *net.UDPConn

	listenCtx    context.Context
	listenCancel context.CancelFunc
	running      bool

	connLock sync.Mutex
	conns    map[string]*Conn
}

func (l *Listener) Accept() (net.Conn, error) {
	// Yeah, this is a hack
	if l.listenCancel == nil {
		l.listenCtx, l.listenCancel = context.WithCancel(context.Background())
		l.running = true
		go l.reader()
	}

	<-l.listenCtx.Done()
	return nil, errors.New("listener closed")
}

var _ net.Listener = &Listener{}

func NewListener(addr string) (*Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		addr:     udpAddr,
		listener: conn,
		conns:    make(map[string]*Conn),
	}
	return l, nil
}

func (l *Listener) removeConn(conn *Conn) {
	l.connLock.Lock()
	defer l.connLock.Unlock()

	delete(l.conns, conn.remoteAddr.String())
}

func (l *Listener) reader() {
	buf := make([]byte, 65535)
	for l.running {
		n, addr, err := l.listener.ReadFromUDP(buf)
		if err != nil {
			return
		}

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

		go conn.handlePacket(buf[:n])
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

	return l.listener.Close()
}
