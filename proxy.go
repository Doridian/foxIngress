package main

import (
	"log"
	"sync"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
	"github.com/Doridian/foxIngress/conn/tcp"
	"github.com/Doridian/foxIngress/conn/udp"
	"github.com/Doridian/foxIngress/util"
)

var initWait sync.WaitGroup
var listenerClosedWait sync.WaitGroup
var privilegeDropWait sync.WaitGroup

func doProxy(host string, protocol config.BackendProtocol) {
	defer func() {
		listenerClosedWait.Done()
		log.Fatalf("listener goroutine ended unexpectedly")
	}()

	listenProto := ""
	var listener conn.Listener
	var err error
	if protocol == config.PROTO_QUIC {
		listener, err = udp.NewListener(host, protocol)
		listenProto = "UDP"
	} else {
		listener, err = tcp.NewListener(host, protocol)
		listenProto = "TCP"
	}

	initWait.Done()
	if err != nil {
		log.Fatalf("could not listen on %s %s: %v", listenProto, host, err)
		return
	}

	log.Printf("Listener started on %s %s", listenProto, host)
	privilegeDropWait.Wait()

	log.Printf("Server started on %s %s", listenProto, host)
	listener.Start()
}

func main() {
	config.Load()

	privilegeDropWait.Add(1)

	initWait.Add(3)
	listenerClosedWait.Add(3)
	go doProxy(config.GetHTTPAddr(), config.PROTO_HTTP)
	go doProxy(config.GetHTTPSAddr(), config.PROTO_HTTPS)
	go doProxy(config.GetQUICAddr(), config.PROTO_QUIC)

	initWait.Wait()
	util.DropPrivs()
	privilegeDropWait.Done()

	listenerClosedWait.Wait()
}
