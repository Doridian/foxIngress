package conn

import "github.com/prometheus/client_golang/prometheus"

type Listener interface {
	Start()
	IPProto() string
}

var RawConnectionsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "foxingress_raw_connections_total",
		Help: "Total number of connections accepted by a listener",
	},
	[]string{"proto", "ipproto", "listener"},
)

var OpenConnections = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "foxingress_open_connections",
		Help: "Number of open connections made to a backend",
	},
	[]string{"proto", "ipproto", "listener", "backend"},
)

var ConnectionsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "foxingress_connections_total",
		Help: "Total number of connections made to a backend",
	},
	[]string{"proto", "ipproto", "listener", "backend"},
)
