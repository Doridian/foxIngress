/*
A simple routing server in Go.  Accepts incoming connections on ports 80 and 443.
*/

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fzzy/radix/redis"
	"github.com/inconshreveable/go-vhost"
)

var defaultBackends map[string]string

func getBackend(hostname string, protocol string, redisClient *redis.Client) (string, error) {
	var backend string
	var err error

	res := redisClient.Cmd("get", "hostnames:"+hostname+":backend")
	if res.Type == redis.NilReply {
		backend = ""
	} else if res.Type == redis.ErrorReply {
		return "", res.Err
	} else {
		backend, err = res.Str()
	}
	if err != nil {
		fmt.Println("Error in redis lookup for hostname backend", err)
		return "", err
	}
	if len(backend) < 1 {
		return defaultBackends[protocol], nil
	}
	return backend, nil
}

func handleHTTPConnection(client net.Conn, redisClient *redis.Client) {
	defer client.Close()

	vhostConn, err := vhost.HTTP(client)
	if err != nil {
		return
	}
	// read out the Host header and auth from the request
	hostname := strings.ToLower(vhostConn.Host())
	client = vhostConn
	vhostConn.Free()
	backend, err := getBackend(hostname, "http", redisClient)
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

func handleHTTPSConnection(client net.Conn, redisClient *redis.Client) {
	defer client.Close()

	vhostConn, err := vhost.TLS(client)
	if err != nil {
		fmt.Println("Could not extract SNI handshake")
		return
	}
	hostname := vhostConn.Host()
	client = vhostConn
	vhostConn.Free()

	backend, err := getBackend(hostname, "https", redisClient)
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
func doProxy(done chan int, host string, handle func(net.Conn, *redis.Client), redisClient *redis.Client) {
	defer reportDone(done)
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

		go handle(connection, redisClient)
	}
}

func main() {
	defaultBackends = make(map[string]string)
	defaultBackends["http"] = os.Getenv("DEFAULT_BACKEND_HTTP")
	defaultBackends["https"] = os.Getenv("DEFAULT_BACKEND_HTTPS")

	redisClient, error := redis.Dial("tcp", os.Getenv("REDIS_HOST"))
	if error != nil {
		fmt.Println("Error connecting to redis", error)
		os.Exit(1)
	}

	httpDone := make(chan int)
	go doProxy(httpDone, ":80", handleHTTPConnection, redisClient)

	httpsDone := make(chan int)
	go doProxy(httpsDone, ":443", handleHTTPSConnection, redisClient)

	<-httpDone
	<-httpsDone
}
