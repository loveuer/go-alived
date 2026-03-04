package health

import (
	"fmt"

	"github.com/loveuer/go-alived/pkg/config"
	"github.com/loveuer/go-alived/pkg/logger"
)

func CreateChecker(cfg *config.HealthChecker) (Checker, error) {
	if cfg.Config == nil {
		return nil, fmt.Errorf("missing config for checker %s", cfg.Name)
	}

	configMap, ok := cfg.Config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config type for checker %s: expected map[string]interface{}", cfg.Name)
	}

	switch cfg.Type {
	case "tcp":
		return NewTCPChecker(cfg.Name, configMap)
	case "http", "https":
		return NewHTTPChecker(cfg.Name, configMap)
	case "ping", "icmp":
		return NewPingChecker(cfg.Name, configMap)
	case "script":
		return NewScriptChecker(cfg.Name, configMap)
	default:
		return nil, fmt.Errorf("unsupported checker type: %s", cfg.Type)
	}
}

func LoadFromConfig(cfg *config.Config, log *logger.Logger) (*Manager, error) {
	manager := NewManager(log)

	for _, healthCfg := range cfg.Health {
		checker, err := CreateChecker(&healthCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create checker %s: %w", healthCfg.Name, err)
		}

		configMap, _ := healthCfg.Config.(map[string]interface{})
		monitorCfg := &CheckerConfig{
			Name:     healthCfg.Name,
			Type:     healthCfg.Type,
			Interval: healthCfg.Interval,
			Timeout:  healthCfg.Timeout,
			Rise:     healthCfg.Rise,
			Fall:     healthCfg.Fall,
			Config:   configMap,
		}

		monitor := NewMonitor(checker, monitorCfg, log)
		manager.AddMonitor(monitor)

		log.Info("loaded health checker: %s (type=%s)", healthCfg.Name, healthCfg.Type)
	}

	return manager, nil
}
