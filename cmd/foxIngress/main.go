package main

import (
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn/reg"
	"github.com/Doridian/foxIngress/util"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var initWait sync.WaitGroup
var listenerClosedWait sync.WaitGroup
var privilegeDropWait sync.WaitGroup

func doProxy(host string, proto config.BackendProtocol) {
	defer func() {
		listenerClosedWait.Done()
		log.Fatalf("listener goroutine ended unexpectedly")
	}()

	listener, err := reg.GetListenerForProto(host, proto)
	ipProto := listener.IPProto()

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

func promListen() {
	promAddr := config.GetPrometheusAddr()
	if promAddr == "" {
		initWait.Done()
		listenerClosedWait.Done()
		return
	}

	ln, err := net.Listen("tcp", promAddr)
	if err != nil {
		log.Fatalf("Error starting Prometheus listener: %v", err)
	}

	log.Printf("Prometheus listener started on %s", promAddr)

	http.Handle("/metrics", promhttp.Handler())

	initWait.Done()
	privilegeDropWait.Wait()

	go func() {
		defer listenerClosedWait.Done()

		err := http.Serve(ln, nil)
		if err != nil {
			log.Fatalf("Error serving Prometheus listener: %v", err)
		}
	}()

	log.Printf("Prometheus listener enabled on %s", promAddr)
}

func main() {
	log.Printf("foxIngress version %s", util.Version)

	config.Load()

	privilegeDropWait.Add(1)

	initWait.Add(4)
	listenerClosedWait.Add(4)
	go promListen()
	go doProxy(config.GetHTTPAddr(), config.PROTO_HTTP)
	go doProxy(config.GetHTTPSAddr(), config.PROTO_HTTPS)
	go doProxy(config.GetQUICAddr(), config.PROTO_QUIC)

	initWait.Wait()
	util.DropPrivs()
	privilegeDropWait.Done()

	listenerClosedWait.Wait()
}
