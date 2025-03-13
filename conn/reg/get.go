package reg

import (
	"fmt"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
	"github.com/Doridian/foxIngress/conn/tcp"
	"github.com/Doridian/foxIngress/conn/udp"
)

func GetListenerForProto(host string, proto config.BackendProtocol) (listener conn.Listener, err error) {
	switch proto {
	case config.PROTO_HTTP, config.PROTO_HTTPS:
		listener, err = tcp.NewListener(host, proto)
	case config.PROTO_QUIC:
		listener, err = udp.NewListener(host, proto)
	default:
		return nil, fmt.Errorf("unknown protocol %v", proto)
	}

	return
}
