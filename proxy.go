package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/FoxDenHome/sni-vhost-proxy/config"
	"github.com/FoxDenHome/sni-vhost-proxy/udpconn"
	"github.com/FoxDenHome/sni-vhost-proxy/util"
	"github.com/inconshreveable/go-vhost"
)

var initWait sync.WaitGroup
var listenerClosedWait sync.WaitGroup
var privilegeDropWait sync.WaitGroup

func handleConnection(client net.Conn, protocol config.BackendProtocol) {
	defer client.Close()

	var vhostConn vhost.Conn
	var err error
	switch protocol {
	case config.PROTO_HTTP:
		vhostConn, err = vhost.HTTP(client)
	case config.PROTO_HTTPS:
		vhostConn, err = vhost.TLS(client)
	default:
		log.Printf("Invalid protocol from %v", client.RemoteAddr())
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
	backend, err := config.GetBackend(hostname, protocol)
	if err != nil {
		log.Printf("Couldn't get backend for %s: %v", hostname, err)
		return
	}

	if backend == nil {
		// This means we don't want to handle the connection
		return
	}

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
	defer upConn.Close()

	if backend.ProxyProtocol {
		data, err := util.MakeProxyProtocolPayload(client.RemoteAddr().(*net.TCPAddr).AddrPort(), client.LocalAddr().(*net.TCPAddr).AddrPort())
		if err != nil {
			log.Printf("Could not make PROXY protocol payload for %s: %v", hostname, err)
			return
		}
		_, err = upConn.Write(data)
		if err != nil {
			log.Printf("Could not write PROXY protocol payload for %s: %v", hostname, err)
			return
		}
	}

	joinConnections(vhostConn, upConn)
}

func halfJoin(wg *sync.WaitGroup, dst net.Conn, src net.Conn) {
	defer wg.Done()
	defer dst.Close()
	defer src.Close()
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
	log.Printf("Conn closed: %v -> %v", c1, c2)
}

func doProxy(host string, protocol config.BackendProtocol) {
	defer func() {
		listenerClosedWait.Done()
		log.Panicf("listener goroutine ended unexpectedly")
	}()

	var listener net.Listener
	var err error
	if protocol == config.PROTO_QUIC {
		listener, err = udpconn.NewListener(host)
	} else {
		listener, err = net.Listen("tcp", host)
	}

	initWait.Done()
	if err != nil {
		log.Panicf("could not listen on %s: %v", host, err)
		return
	}

	log.Printf("Listener started on %s", host)
	privilegeDropWait.Wait()

	log.Printf("Server started on %s", host)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			return
		}

		go handleConnection(connection, protocol)
	}
}

func main() {
	config.LoadConfig()

	privilegeDropWait.Add(1)

	initWait.Add(3)
	listenerClosedWait.Add(3)
	go doProxy(os.Getenv("HTTP_ADDR"), config.PROTO_HTTP)
	go doProxy(os.Getenv("HTTPS_ADDR"), config.PROTO_HTTPS)
	go doProxy(os.Getenv("QUIC_ADDR"), config.PROTO_QUIC)

	initWait.Wait()
	util.DropPrivs()
	privilegeDropWait.Done()

	listenerClosedWait.Wait()
}
