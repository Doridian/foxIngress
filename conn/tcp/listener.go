package tcp

import (
	"errors"
	"log"
	"net"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
)

type Listener struct {
	listener net.Listener
	proto    config.BackendProtocol
}

var _ conn.Listener = &Listener{}

func NewListener(host string, proto config.BackendProtocol) (*Listener, error) {
	if proto == config.PROTO_QUIC {
		return nil, errors.New("TCP listener does not support QUIC")
	}

	listener, err := net.Listen("tcp", host)
	if err != nil {
		return nil, err
	}

	return &Listener{
		listener: listener,
		proto:    proto,
	}, nil
}

func (l *Listener) Start() {
	for {
		connection, err := l.listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			return
		}

		go l.handleConnection(connection)
	}
}
