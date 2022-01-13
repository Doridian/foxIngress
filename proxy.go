/*
A simple routing server in Go.  Accepts incoming connections on ports 80 and 443.
*/

package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/go-vhost"
)

func handleHTTPConnection(client net.Conn) {
	defer client.Close()

	vhostConn, err := vhost.HTTP(client)
	if err != nil {
		return
	}
	// read out the Host header and auth from the request
	hostname := strings.ToLower(vhostConn.Host())
	client = vhostConn
	vhostConn.Free()
	backend, err := GetBackend(hostname, "http")
	if err != nil || backend == "" {
		fmt.Println("Couldn't get backend for ", hostname, "-- got error", err)
		return
	}
	upConn, err := net.DialTimeout("tcp", backend, time.Duration(10000)*time.Millisecond)
	if err != nil {
		fmt.Printf("Failed to dial backend connection %v: %v\n", backend, err)
		return
	}
	fmt.Printf("Initiated new connection to backend: %v %v\n", upConn.LocalAddr(), upConn.RemoteAddr())

	joinConnections(client, upConn)
}

func handleHTTPSConnection(client net.Conn) {
	defer client.Close()

	vhostConn, err := vhost.TLS(client)
	if err != nil {
		fmt.Println("Could not extract SNI handshake")
		return
	}
	hostname := vhostConn.Host()
	client = vhostConn
	vhostConn.Free()

	backend, err := GetBackend(hostname, "https")
	if err != nil {
		fmt.Println("Couldn't get backend for ", hostname, "-- got error", err)
		return
	}

	upConn, err := net.DialTimeout("tcp", backend, time.Duration(10000)*time.Millisecond)
	if err != nil {
		fmt.Printf("Failed to dial backend connection %v: %v\n", backend, err)
		return
	}
	fmt.Printf("Initiated new connection to backend: %v %v\n", upConn.LocalAddr(), upConn.RemoteAddr())

	joinConnections(client, upConn)
}

func halfJoin(wg sync.WaitGroup, dst net.Conn, src net.Conn) {
	defer wg.Done()
	defer dst.Close()
	defer src.Close()
	n, err := io.Copy(dst, src)
	fmt.Printf("Copy from %v to %v failed after %d bytes with error %v\n", src.RemoteAddr(), dst.RemoteAddr(), n, err)
}
func joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	fmt.Printf("Joining connections: %v %v\n", c1.RemoteAddr(), c2.RemoteAddr())
	wg.Add(2)
	go halfJoin(wg, c1, c2)
	go halfJoin(wg, c2, c1)
	wg.Wait()
}

func doProxy(done chan int, host string, handle func(net.Conn)) {
	defer func() {
		done <- 1
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
