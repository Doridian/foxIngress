package reg

import (
	"fmt"

	"github.com/Doridian/foxIngress/config"
	"github.com/Doridian/foxIngress/conn"
	"github.com/Doridian/foxIngress/conn/tcp"
	"github.com/Doridian/foxIngress/conn/udp"
)

func GetListenerForProto(host string, proto config.BackendProtocol) (conn.Listener, error) {
	switch proto {
	case config.PROTO_HTTP, config.PROTO_HTTPS:
		return tcp.NewListener(host, proto)
	case config.PROTO_QUIC:
		return udp.NewListener(host, proto)
	default:
		return nil, fmt.Errorf("unknown protocol %v", proto)
	}
}
