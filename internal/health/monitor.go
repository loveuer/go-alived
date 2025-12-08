package health

import (
	"context"
	"sync"
	"time"

	"github.com/loveuer/go-alived/pkg/logger"
)

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
	callbacks := m.callbacks
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

func (m *Monitor) OnStateChange(callback StateChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

func (m *Monitor) GetState() *CheckerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stateCopy := *m.state
	return &stateCopy
}

func (m *Monitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.Healthy
}

type Manager struct {
	monitors map[string]*Monitor
	mu       sync.RWMutex
	log      *logger.Logger
}

func NewManager(log *logger.Logger) *Manager {
	return &Manager{
		monitors: make(map[string]*Monitor),
		log:      log,
	}
}

func (m *Manager) AddMonitor(monitor *Monitor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.monitors[monitor.config.Name] = monitor
}

func (m *Manager) GetMonitor(name string) (*Monitor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	monitor, ok := m.monitors[name]
	return monitor, ok
}

func (m *Manager) StartAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, monitor := range m.monitors {
		monitor.Start()
	}

	m.log.Info("started %d health check monitor(s)", len(m.monitors))
}

func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, monitor := range m.monitors {
		monitor.Stop()
	}

	m.log.Info("stopped all health check monitors")
}

func (m *Manager) GetAllStates() map[string]*CheckerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]*CheckerState)
	for name, monitor := range m.monitors {
		states[name] = monitor.GetState()
	}

	return states
}
