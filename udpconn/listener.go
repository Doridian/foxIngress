package udpconn

import (
	"context"
	"net"
	"sync"
)

type Listener struct {
	addr     *net.UDPAddr
	listener *net.UDPConn

	listenCtx    context.Context
	listenCancel context.CancelFunc
	running      bool

	connLock  sync.Mutex
	conns     map[string]*Conn
	connQueue chan *Conn
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
		addr:      udpAddr,
		listener:  conn,
		conns:     make(map[string]*Conn),
		connQueue: make(chan *Conn, 1024),
		running:   true,
	}
	l.listenCtx, l.listenCancel = context.WithCancel(context.Background())
	go l.reader()
	return l, nil
}

func (l *Listener) removeConn(conn *Conn) {
	l.connLock.Lock()
	defer l.connLock.Unlock()

	delete(l.conns, conn.remoteAddr.String())
}

func (l *Listener) reader() {
	for l.running {
		buf := make([]byte, 4096)
		_, addr, err := l.listener.ReadFromUDP(buf)
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
			}
			l.conns[connKey] = conn
		}
		l.connLock.Unlock()

		conn.handlePacket(buf)

		<-l.connQueue
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	for {
		select {
		case conn := <-l.connQueue:
			return conn, nil
		case <-l.listenCtx.Done():
			return nil, l.listenCtx.Err()
		}
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
