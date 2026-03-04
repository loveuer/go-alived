package health

import (
	"context"
	"sync"
	"time"

	"github.com/loveuer/go-alived/pkg/logger"
)

// Monitor runs periodic health checks and tracks state.
type Monitor struct {
	checker   Checker
	config    *CheckerConfig
	state     *CheckerState
	log       *logger.Logger
	callbacks []StateChangeCallback

	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.RWMutex
}

// NewMonitor creates a new Monitor for the given checker.
func NewMonitor(checker Checker, config *CheckerConfig, log *logger.Logger) *Monitor {
	return &Monitor{
		checker: checker,
		config:  config,
		state: &CheckerState{
			Name:    config.Name,
			Healthy: false,
		},
		log:       log,
		callbacks: make([]StateChangeCallback, 0),
		stopCh:    make(chan struct{}),
	}
}

// Start begins the health check loop.
func (m *Monitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.log.Info("[HealthCheck:%s] starting health check monitor (interval=%s, timeout=%s)",
		m.config.Name, m.config.Interval, m.config.Timeout)

	m.wg.Add(1)
	go m.checkLoop()
}

// Stop stops the health check loop.
func (m *Monitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	m.log.Info("[HealthCheck:%s] stopping health check monitor", m.config.Name)
	close(m.stopCh)
	m.wg.Wait()
}

func (m *Monitor) checkLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Perform initial check immediately
	m.performCheck()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.performCheck()
		}
	}
}

func (m *Monitor) performCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	startTime := time.Now()
	result := m.checker.Check(ctx)
	duration := time.Since(startTime)

	m.mu.Lock()
	oldHealthy := m.state.Healthy
	stateChanged := m.state.Update(result, m.config.Rise, m.config.Fall)
	newHealthy := m.state.Healthy
	callbacks := make([]StateChangeCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.Unlock()

	m.log.Debug("[HealthCheck:%s] check completed: result=%s, duration=%s, healthy=%v",
		m.config.Name, result, duration, newHealthy)

	if stateChanged {
		m.log.Info("[HealthCheck:%s] health state changed: %v -> %v (consecutive_ok=%d, consecutive_fail=%d)",
			m.config.Name, oldHealthy, newHealthy, m.state.ConsecutiveOK, m.state.ConsecutiveFail)

		for _, callback := range callbacks {
			callback(m.config.Name, oldHealthy, newHealthy)
		}
	}
}

// OnStateChange registers a callback for health state changes.
func (m *Monitor) OnStateChange(callback StateChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// GetState returns a copy of the current checker state.
func (m *Monitor) GetState() *CheckerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stateCopy := *m.state
	return &stateCopy
}

// IsHealthy returns whether the checker is currently healthy.
func (m *Monitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.Healthy
}
