package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/go-vhost"
)

func handleConnection(client net.Conn, protocol BackendProtocol) {
	defer client.Close()

	var vhostConn vhost.Conn
	var err error
	switch protocol {
	case PROTO_HTTP:
		vhostConn, err = vhost.HTTP(client)
	case PROTO_HTTPS:
		vhostConn, err = vhost.TLS(client)
	default:
		log.Printf("Invalid protocol from %v", client.RemoteAddr())
		return
	}
	if err != nil {
		log.Printf("Error decoding protocol from %v: %v", client.RemoteAddr(), err)
		return
	}

	hostname := strings.ToLower(vhostConn.Host())
	vhostConn.Free()
	backend, err := GetBackend(hostname, protocol)
	if err != nil || backend == "" {
		log.Printf("Couldn't get backend for %s: %v", hostname, err)
		return
	}
	upConn, err := net.DialTimeout("tcp", backend, time.Duration(10000)*time.Millisecond)
	if err != nil {
		log.Printf("Couldn't dial backend connection for %s: %v", hostname, err)
		return
	}

	joinConnections(vhostConn, upConn)
}

func handleHTTPConnection(client net.Conn) {
	handleConnection(client, PROTO_HTTP)
}

func handleHTTPSConnection(client net.Conn) {
	handleConnection(client, PROTO_HTTPS)
}

func halfJoin(wg sync.WaitGroup, dst net.Conn, src net.Conn) {
	defer wg.Done()
	defer dst.Close()
	defer src.Close()
	_, err := io.Copy(dst, src)
	if err == nil || errors.Is(err, net.ErrClosed) {
		return
	}
	log.Printf("Proxy copy from %v to %v failed with error %v", src.RemoteAddr(), dst.RemoteAddr(), err)
}
func joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go halfJoin(wg, c1, c2)
	go halfJoin(wg, c2, c1)
	wg.Wait()
}

func doProxy(done chan int, host string, handle func(net.Conn)) {
	defer func() {
		done <- 1
		log.Panicf("listener goroutine ended unexpectedly")
	}()
	listener, err := net.Listen("tcp", host)
	if err != nil {
		log.Panicf("could not listen: %v", err)
		return
	}
	log.Printf("Server started on %s", host)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			return
		}

		go handle(connection)
	}
}

func main() {
	LoadConfig()

	httpDone := make(chan int)
	go doProxy(httpDone, ":80", handleHTTPConnection)

	httpsDone := make(chan int)
	go doProxy(httpsDone, ":443", handleHTTPSConnection)

	<-httpDone
	<-httpsDone
}
