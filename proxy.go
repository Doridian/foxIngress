/*
A simple routing server in Go.  Accepts incoming connections on ports 80 and 443.
*/

package main

import (
	"errors"
	"fmt"
	"io"
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
		fmt.Println("Invalid protocol!")
		return
	}
	if err != nil {
		fmt.Println("Error decoding protocol:", err)
		return
	}

	hostname := strings.ToLower(vhostConn.Host())
	vhostConn.Free()
	backend, err := GetBackend(hostname, protocol)
	if err != nil || backend == "" {
		fmt.Println("Couldn't get backend for ", hostname, "-- got error", err)
		return
	}
	upConn, err := net.DialTimeout("tcp", backend, time.Duration(10000)*time.Millisecond)
	if err != nil {
		fmt.Printf("Failed to dial backend connection %v: %v\n", backend, err)
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
	n, err := io.Copy(dst, src)
	if err == nil || errors.Is(err, net.ErrClosed) {
		return
	}
	fmt.Printf("Copy from %v to %v failed after %d bytes with error %v\n", src.RemoteAddr(), dst.RemoteAddr(), n, err)
}
func joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	//fmt.Printf("Joining connections: %v %v\n", c1.RemoteAddr(), c2.RemoteAddr())
	wg.Add(2)
	go halfJoin(wg, c1, c2)
	go halfJoin(wg, c2, c1)
	wg.Wait()
}

func doProxy(done chan int, host string, handle func(net.Conn)) {
	defer func() {
		done <- 1
		panic(errors.New("listeners should never end: panic"))
	}()
	listener, err := net.Listen("tcp", host)
	if err != nil {
		fmt.Println("Couldn't start listening", err)
		return
	}
	fmt.Println("Started server on", host, "-- listening...")
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error", err)
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
