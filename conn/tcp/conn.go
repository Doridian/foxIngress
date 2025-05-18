package tcp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
	"github.com/Doridian/foxIngress/util/proxy"
	"github.com/inconshreveable/go-vhost"
)

func (l *Listener) handleConnection(client net.Conn) {
	conn.RawConnectionsTotal.WithLabelValues(l.proto.String(), l.IPProto(), l.listener.Addr().String()).Inc()

	defer func() {
		_ = client.Close()
	}()

	var vhostConn vhost.Conn
	var err error
	switch l.proto {
	case config.PROTO_HTTP:
		vhostConn, err = vhost.HTTP(client)
	case config.PROTO_HTTPS:
		vhostConn, err = vhost.TLS(client)
	default:
		log.Fatalf("Invalid TCP protocol %s", l.proto.String())
		return
	}
	if err != nil {
		if config.Verbose {
			log.Printf("Error decoding protocol from %v: %v", client.RemoteAddr(), err)
		}
		return
	}

	hostname := strings.ToLower(vhostConn.Host())
	vhostConn.Free()
	backend, err := config.GetBackend(hostname, l.proto)
	if err != nil {
		log.Printf("Couldn't get backend for %s: %v", hostname, err)
		return
	}

	if backend == nil {
		// This means we don't want to handle the connection
		return
	}

	conn.OpenConnections.WithLabelValues(l.proto.String(), l.IPProto(), l.listener.Addr().String(), backend.Match, backend.String()).Inc()
	conn.ConnectionsTotal.WithLabelValues(l.proto.String(), l.IPProto(), l.listener.Addr().String(), backend.Match, backend.String()).Inc()
	defer conn.OpenConnections.WithLabelValues(l.proto.String(), l.IPProto(), l.listener.Addr().String(), backend.Match, backend.String()).Dec()

	useHost := backend.Host
	if backend.HostPassthrough {
		useHost = hostname
	}

	ipport := fmt.Sprintf("[%s]:%d", useHost, backend.Port)
	upConn, err := net.DialTimeout("tcp", ipport, time.Duration(10000)*time.Millisecond)
	if err != nil {
		log.Printf("Couldn't dial backend connection for %s: %v", hostname, err)
		return
	}
	defer func() {
		_ = upConn.Close()
	}()

	if backend.ProxyProtocol {
		err = proxy.WriteConn(upConn)
		if err != nil {
			log.Printf("Could not write PROXY protocol payload for %s: %v", hostname, err)
			return
		}
	}

	joinConnections(vhostConn, upConn)
}

func halfJoin(wg *sync.WaitGroup, dst net.Conn, src net.Conn) {
	defer func() {
		wg.Done()
		_ = dst.Close()
		_ = src.Close()
	}()

	_, err := io.Copy(dst, src)
	if err == nil || errors.Is(err, net.ErrClosed) {
		return
	}
	if config.Verbose {
		log.Printf("Proxy copy from %v to %v failed with error %v", src.RemoteAddr(), dst.RemoteAddr(), err)
	}
}

func joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go halfJoin(&wg, c1, c2)
	go halfJoin(&wg, c2, c1)
	wg.Wait()
}
