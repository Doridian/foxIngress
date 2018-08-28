/*
A simple routing proxy in Go.  Accepts incoming connections on ports 80 and 443.
*/

package main

import (
	"errors"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/inconshreveable/go-vhost"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func getBackend(hostname string, defaultBackendType string, consulClient *api.KV) (string, error) {
	fmt.Println("Looking up", hostname)
	// Lookup the pair
	pair, _, err := consulClient.Get(defaultBackendType+hostname, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	if pair == nil {
		return "", errors.New("No hostname found")
	}
	fmt.Println("Found backends:", string(pair.Value))
	return string(pair.Value), nil
}

func handleHTTPConnection(client net.Conn, consulClient *api.KV) {
	vhostConn, err := vhost.HTTP(client)
	if err != nil {
		return
	}
	// read out the Host header and auth from the request
	hostname := strings.ToLower(vhostConn.Host())
	client = vhostConn
	vhostConn.Free()
	backend, err := getBackend(hostname, "http/", consulClient)
	if err != nil {
		fmt.Println("Couldn't get backend for ", hostname, "-- got error", err)
		client.Close()
		return
	}
	upConn, err := net.DialTimeout("tcp", backend, time.Duration(10000)*time.Millisecond)
	if err != nil {
		fmt.Printf("Failed to dial backend connection %v: %v\n", backend, err)
		client.Close()
		return
	}
	fmt.Printf("Initiated new connection to backend: %v %v\n", upConn.LocalAddr(), upConn.RemoteAddr())

	// join the connections
	joinConnections(client, upConn)
	return
}

func handleHTTPSConnection(client net.Conn, consulClient *api.KV) {
	vhostConn, err := vhost.TLS(client)
	hostname := vhostConn.Host()
	client = vhostConn
	vhostConn.Free()

	backend, err := getBackend(hostname, "https/", consulClient)
	if err != nil {
		fmt.Println("Couldn't get backend for ", hostname, "-- got error", err)
		client.Close()
		return
	}

	upConn, err := net.DialTimeout("tcp", backend, time.Duration(10000)*time.Millisecond)
	if err != nil {
		fmt.Printf("Failed to dial backend connection %v: %v\n", backend, err)
		client.Close()
		return
	}
	fmt.Printf("Initiated new connection to backend: %v %v\n", upConn.LocalAddr(), upConn.RemoteAddr())

	// join the connections
	joinConnections(client, upConn)
	return
}

func joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	halfJoin := func(dst net.Conn, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		n, err := io.Copy(dst, src)
		fmt.Printf("Copy from %v to %v failed after %d bytes with error %v\n", src.RemoteAddr(), dst.RemoteAddr(), n, err)
	}

	fmt.Printf("Joining connections: %v %v\n", c1.RemoteAddr(), c2.RemoteAddr())
	wg.Add(2)
	go halfJoin(c1, c2)
	go halfJoin(c2, c1)
	wg.Wait()
}

func reportDone(done chan int) {
	done <- 1
}
func doProxy(done chan int, port int, handle func(net.Conn, *api.KV), consulClient *api.KV) {
	defer reportDone(done)
	listener, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Couldn't start listening", err)
		return
	}
	fmt.Println("Started proxy on", port, "-- listening...")
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error", err)
			return
		}

		go handle(connection, consulClient)
	}
}

func main() {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// Get a handle to the KV API
	kv := client.KV()

	httpDone := make(chan int)
	go doProxy(httpDone, 3999, handleHTTPConnection, kv)

	httpsDone := make(chan int)
	go doProxy(httpsDone, 4000, handleHTTPSConnection, kv)

	<-httpDone
	<-httpsDone
}
