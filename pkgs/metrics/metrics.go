package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type OperationMetrics struct {
	TotalRequests   atomic.Int64
	SuccessRequests atomic.Int64
	FailedRequests  atomic.Int64
	TotalDuration   atomic.Int64
}

type UpstreamMetrics struct {
	TotalRequests   atomic.Int64
	SuccessRequests atomic.Int64
	FailedRequests  atomic.Int64
	TotalLatency    atomic.Int64
}

type Metrics struct {
	mu             sync.RWMutex
	operations     map[string]*OperationMetrics
	upstreams      map[string]*UpstreamMetrics
	startTime      time.Time
	totalRequests  atomic.Int64
	activeRequests atomic.Int64
}

func New() *Metrics {
	return &Metrics{
		operations: make(map[string]*OperationMetrics),
		upstreams:  make(map[string]*UpstreamMetrics),
		startTime:  time.Now(),
	}
}

func (m *Metrics) RecordRequest(operation string, duration time.Duration, success bool) {
	m.mu.RLock()
	opMetrics, exists := m.operations[operation]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		opMetrics = &OperationMetrics{}
		m.operations[operation] = opMetrics
		m.mu.Unlock()
	}

	opMetrics.TotalRequests.Add(1)
	if success {
		opMetrics.SuccessRequests.Add(1)
	} else {
		opMetrics.FailedRequests.Add(1)
	}
	opMetrics.TotalDuration.Add(duration.Microseconds())
	m.totalRequests.Add(1)
}

func (m *Metrics) RecordUpstreamRequest(url string, latency time.Duration, success bool) {
	m.mu.RLock()
	upMetrics, exists := m.upstreams[url]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		upMetrics = &UpstreamMetrics{}
		m.upstreams[url] = upMetrics
		m.mu.Unlock()
	}

	upMetrics.TotalRequests.Add(1)
	if success {
		upMetrics.SuccessRequests.Add(1)
	} else {
		upMetrics.FailedRequests.Add(1)
	}
	upMetrics.TotalLatency.Add(latency.Microseconds())
}

func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"uptime_seconds":  time.Since(m.startTime).Seconds(),
		"total_requests":  m.totalRequests.Load(),
		"operations":      make(map[string]interface{}),
		"upstreams":       make(map[string]interface{}),
		"active_requests": m.activeRequests.Load(),
	}

	for op, metrics := range m.operations {
		stats["operations"].(map[string]interface{})[op] = map[string]interface{}{
			"total":    metrics.TotalRequests.Load(),
			"success":  metrics.SuccessRequests.Load(),
			"failed":   metrics.FailedRequests.Load(),
			"avg_time": float64(metrics.TotalDuration.Load()) / float64(metrics.TotalRequests.Load()) / 1000.0,
		}
	}

	for url, metrics := range m.upstreams {
		stats["upstreams"].(map[string]interface{})[url] = map[string]interface{}{
			"total":       metrics.TotalRequests.Load(),
			"success":     metrics.SuccessRequests.Load(),
			"failed":      metrics.FailedRequests.Load(),
			"avg_latency": float64(metrics.TotalLatency.Load()) / float64(metrics.TotalRequests.Load()) / 1000.0,
		}
	}

	return stats
}

func (m *Metrics) IncActiveRequests() {
	m.activeRequests.Add(1)
}

func (m *Metrics) DecActiveRequests() {
	m.activeRequests.Add(-1)
}
