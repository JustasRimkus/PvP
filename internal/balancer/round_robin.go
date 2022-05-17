package balancer

import "sync"

type roundRobin struct {
	mu      sync.Mutex
	next    int
	targets []instance
	metrics *metrics
}

func (rr *roundRobin) Pick() (string, func()) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	id := rr.next
	target := rr.targets[id]

	target.connections++
	rr.targets[id] = target
	rr.metrics.activeConnections[target.addr].Inc()
	rr.metrics.totalConnections[target.addr].Inc()

	rr.next = rr.next + 1
	if len(rr.targets) == rr.next {
		rr.next = 0
	}

	return target.addr, func() {
		rr.mu.Lock()
		defer rr.mu.Unlock()

		target := rr.targets[id]
		target.connections--
		rr.targets[id] = target
		rr.metrics.activeConnections[target.addr].Dec()
	}
}
