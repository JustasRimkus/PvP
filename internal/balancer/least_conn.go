package balancer

import "sync"

type leastConn struct {
	mu      sync.Mutex
	targets []instance
	metrics *metrics
}

func (lc *leastConn) Pick() (string, func()) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var min int
	for i := 1; i < len(lc.targets); i++ {
		if lc.targets[min].connections > lc.targets[i].connections {
			min = i
		}
	}

	target := lc.targets[min]
	target.connections++
	lc.metrics.activeConnections[target.addr].Inc()
	lc.metrics.totalConnections[target.addr].Inc()
	lc.targets[min] = target

	return target.addr, func() {
		lc.mu.Lock()
		defer lc.mu.Unlock()

		target := lc.targets[min]
		target.connections--
		lc.targets[min] = target
		lc.metrics.activeConnections[target.addr].Dec()
	}
}
