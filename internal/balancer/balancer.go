package balancer

import (
	"errors"
	"fmt"

	"github.com/JustasRimkus/PvP/internal/core"
	"github.com/prometheus/client_golang/prometheus"
)

type Type string

const (
	TypeRoundRobin Type = "round-robin"
	TypeLeastConn  Type = "least-conn"
)

type Balancer interface {
	Pick() (string, func())
}

type metrics struct {
	activeConnections map[string]prometheus.Gauge
}

type instance struct {
	addr        string
	connections int
}

func New(typ Type, targetAddrs []string) (Balancer, error) {
	if len(targetAddrs) == 0 {
		return nil, errors.New("invalid count of target addresses")
	}

	var (
		targets []instance
		metrics = &metrics{
			activeConnections: make(map[string]prometheus.Gauge),
		}
	)

	for _, targetAddr := range targetAddrs {
		metrics.activeConnections[targetAddr] = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: core.MetricsNamespace,
			Subsystem: core.MetricsSubsystem,
			Name:      fmt.Sprintf("active_connections_%s", targetAddr),
		})

		prometheus.MustRegister(metrics.activeConnections[targetAddr])
		targets = append(targets, instance{
			addr: targetAddr,
		})
	}

	switch typ {
	case TypeRoundRobin:
		return &roundRobin{
			targets: targets,
			metrics: metrics,
		}, nil
	case TypeLeastConn:
		return &leastConn{
			targets: targets,
			metrics: metrics,
		}, nil
	default:
		return nil, errors.New("invalid load balancer type")
	}
}
