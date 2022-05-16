package balancer

import (
	"math/rand"
	"sync"
)

type random struct {
	mu      sync.Mutex
	rand    *rand.Rand
	targets []instance
	metrics *metrics
}

func (r *random) Pick() (string, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.rand.Intn(len(r.targets))
	target := r.targets[id]

	target.connections++
	r.targets[id] = target
	r.metrics.activeConnections[target.addr].Inc()

	return target.addr, func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		target := r.targets[id]
		target.connections--
		r.targets[id] = target
		r.metrics.activeConnections[target.addr].Dec()
	}
}
