package vrrp

import (
	"fmt"
	"sync"

	"github.com/loveuer/go-alived/pkg/config"
	"github.com/loveuer/go-alived/pkg/logger"
)

type Manager struct {
	instances map[string]*Instance
	mu        sync.RWMutex
	log       *logger.Logger
}

func NewManager(log *logger.Logger) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		log:       log,
	}
}

func (m *Manager) LoadFromConfig(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, vrrpCfg := range cfg.VRRP {
		inst, err := NewInstance(
			vrrpCfg.Name,
			uint8(vrrpCfg.VirtualRouterID),
			uint8(vrrpCfg.Priority),
			uint8(vrrpCfg.AdvertInterval),
			vrrpCfg.Interface,
			vrrpCfg.VirtualIPs,
			vrrpCfg.AuthType,
			vrrpCfg.AuthPass,
			vrrpCfg.TrackScripts,
			m.log,
		)
		if err != nil {
			return fmt.Errorf("failed to create instance %s: %w", vrrpCfg.Name, err)
		}

		m.instances[vrrpCfg.Name] = inst
		m.log.Info("loaded VRRP instance: %s", vrrpCfg.Name)
	}

	return nil
}

func (m *Manager) StartAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, inst := range m.instances {
		if err := inst.Start(); err != nil {
			return fmt.Errorf("failed to start instance %s: %w", name, err)
		}
	}

	m.log.Info("started %d VRRP instance(s)", len(m.instances))
	return nil
}

func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inst := range m.instances {
		inst.Stop()
	}

	m.log.Info("stopped all VRRP instances")
}

func (m *Manager) GetInstance(name string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.instances[name]
	return inst, ok
}

func (m *Manager) GetAllInstances() []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		result = append(result, inst)
	}

	return result
}

func (m *Manager) Reload(cfg *config.Config) error {
	m.log.Info("reloading VRRP configuration...")

	m.StopAll()

	m.mu.Lock()
	m.instances = make(map[string]*Instance)
	m.mu.Unlock()

	if err := m.LoadFromConfig(cfg); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := m.StartAll(); err != nil {
		return fmt.Errorf("failed to start instances: %w", err)
	}

	m.log.Info("VRRP configuration reloaded successfully")
	return nil
}