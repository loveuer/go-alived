package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Global Global          `yaml:"global"`
	VRRP   []VRRPInstance  `yaml:"vrrp_instances"`
	Health []HealthChecker `yaml:"health_checkers"`
}

type Global struct {
	RouterID         string `yaml:"router_id"`
	NotificationMail string `yaml:"notification_email"`
}

type VRRPInstance struct {
	Name            string   `yaml:"name"`
	Interface       string   `yaml:"interface"`
	State           string   `yaml:"state"`
	VirtualRouterID int      `yaml:"virtual_router_id"`
	Priority        int      `yaml:"priority"`
	VirtualIPs      []string `yaml:"virtual_ips"`
	AdvertInterval  int      `yaml:"advert_interval"`
	AuthType        string   `yaml:"auth_type"`
	AuthPass        string   `yaml:"auth_pass"`
	NotifyMaster    string   `yaml:"notify_master"`
	NotifyBackup    string   `yaml:"notify_backup"`
	NotifyFault     string   `yaml:"notify_fault"`
	TrackScripts    []string `yaml:"track_scripts"`
}

type HealthChecker struct {
	Name     string        `yaml:"name"`
	Type     string        `yaml:"type"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Rise     int           `yaml:"rise"`
	Fall     int           `yaml:"fall"`
	Config   interface{}   `yaml:"config"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Global.RouterID == "" {
		return fmt.Errorf("global.router_id is required")
	}

	for i, vrrp := range cfg.VRRP {
		if vrrp.Name == "" {
			return fmt.Errorf("vrrp_instances[%d].name is required", i)
		}
		if vrrp.Interface == "" {
			return fmt.Errorf("vrrp_instances[%d].interface is required", i)
		}
		if vrrp.VirtualRouterID < 1 || vrrp.VirtualRouterID > 255 {
			return fmt.Errorf("vrrp_instances[%d].virtual_router_id must be between 1 and 255", i)
		}
		if vrrp.Priority < 1 || vrrp.Priority > 255 {
			return fmt.Errorf("vrrp_instances[%d].priority must be between 1 and 255", i)
		}
		if len(vrrp.VirtualIPs) == 0 {
			return fmt.Errorf("vrrp_instances[%d].virtual_ips cannot be empty", i)
		}
	}

	return nil
}
