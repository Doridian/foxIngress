package main

import (
	"log"
	"sync"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn/reg"
	"github.com/Doridian/foxIngress/util"
)

var initWait sync.WaitGroup
var listenerClosedWait sync.WaitGroup
var privilegeDropWait sync.WaitGroup

func doProxy(host string, proto config.BackendProtocol) {
	defer func() {
		listenerClosedWait.Done()
		log.Fatalf("listener goroutine ended unexpectedly")
	}()

	listener, ipProto, err := reg.GetListenerForProto(host, proto)

	initWait.Done()
	if err != nil {
		log.Fatalf("%s server vcould not listen on %s %s: %v", proto.String(), ipProto, host, err)
		return
	}

	log.Printf("%s listener started on %s %s", proto.String(), ipProto, host)
	privilegeDropWait.Wait()

	log.Printf("%s server started on %s %s", proto.String(), ipProto, host)
	listener.Start()
}

func main() {
	log.Printf("foxIngress version %s", util.Version)

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
