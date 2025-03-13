package conn

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Listener interface {
	Start()
	IPProto() string
}

var RawConnectionsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "foxingress_raw_connections_total",
		Help: "Total number of connections accepted by a listener",
	},
	[]string{"proto", "ipproto", "listener"},
)

var OpenConnections = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "foxingress_open_connections",
		Help: "Number of open connections made to a backend",
	},
	[]string{"proto", "ipproto", "listener", "backend"},
)

var ConnectionsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "foxingress_connections_total",
		Help: "Total number of connections made to a backend",
	},
	[]string{"proto", "ipproto", "listener", "backend"},
)
