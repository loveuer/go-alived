package health

import (
	"sync"

	"github.com/loveuer/go-alived/pkg/logger"
)

// Manager manages multiple health check monitors.
type Manager struct {
	monitors map[string]*Monitor
	mu       sync.RWMutex
	log      *logger.Logger
}

// NewManager creates a new health check Manager.
func NewManager(log *logger.Logger) *Manager {
	return &Manager{
		monitors: make(map[string]*Monitor),
		log:      log,
	}
}

// AddMonitor adds a monitor to the manager.
func (m *Manager) AddMonitor(monitor *Monitor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.monitors[monitor.config.Name] = monitor
}

// GetMonitor retrieves a monitor by name.
func (m *Manager) GetMonitor(name string) (*Monitor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	monitor, ok := m.monitors[name]
	return monitor, ok
}

// StartAll starts all registered monitors.
func (m *Manager) StartAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, monitor := range m.monitors {
		monitor.Start()
	}

	m.log.Info("started %d health check monitor(s)", len(m.monitors))
}

// StopAll stops all registered monitors.
func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, monitor := range m.monitors {
		monitor.Stop()
	}

	m.log.Info("stopped all health check monitors")
}

// GetAllStates returns the current state of all monitors.
func (m *Manager) GetAllStates() map[string]*CheckerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]*CheckerState)
	for name, monitor := range m.monitors {
		states[name] = monitor.GetState()
	}

	return states
}
