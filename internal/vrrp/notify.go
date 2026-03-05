package vrrp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loveuer/go-alived/pkg/logger"
)

const notifyTimeout = 60 * time.Second

// NotifyConfig holds the notify script configuration for a VRRP instance.
type NotifyConfig struct {
	Name          string
	NotifyMaster  string
	NotifyBackup  string
	NotifyFault   string
	Log           *logger.Logger
}

// SetupNotify registers notify scripts as state change callbacks on the instance.
func SetupNotify(inst *Instance, cfg *NotifyConfig) {
	if cfg.NotifyMaster != "" {
		script := cfg.NotifyMaster
		inst.OnMaster(func() {
			cfg.Log.Info("[%s] executing notify_master script", cfg.Name)
			go runNotifyScript(cfg.Log, cfg.Name, "notify_master", script)
		})
		cfg.Log.Info("[%s] registered notify_master script", cfg.Name)
	}

	if cfg.NotifyBackup != "" {
		script := cfg.NotifyBackup
		inst.OnBackup(func() {
			cfg.Log.Info("[%s] executing notify_backup script", cfg.Name)
			go runNotifyScript(cfg.Log, cfg.Name, "notify_backup", script)
		})
		cfg.Log.Info("[%s] registered notify_backup script", cfg.Name)
	}

	if cfg.NotifyFault != "" {
		script := cfg.NotifyFault
		inst.OnFault(func() {
			cfg.Log.Info("[%s] executing notify_fault script", cfg.Name)
			go runNotifyScript(cfg.Log, cfg.Name, "notify_fault", script)
		})
		cfg.Log.Info("[%s] registered notify_fault script", cfg.Name)
	}
}

func runNotifyScript(log *logger.Logger, instName, event, script string) {
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	cmd := buildCommand(ctx, script)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GO_ALIVED_INSTANCE=%s", instName),
		fmt.Sprintf("GO_ALIVED_EVENT=%s", event),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("[%s] %s script failed: %v (output: %s)",
			instName, event, err, strings.TrimSpace(string(output)))
		return
	}

	if len(output) > 0 {
		log.Info("[%s] %s script output: %s",
			instName, event, strings.TrimSpace(string(output)))
	}
	log.Info("[%s] %s script completed successfully", instName, event)
}

func buildCommand(ctx context.Context, script string) *exec.Cmd {
	script = strings.TrimSpace(script)

	// If the script is a path to an existing executable file, run it directly
	if info, err := os.Stat(script); err == nil && !info.IsDir() {
		return exec.CommandContext(ctx, script)
	}

	// Otherwise treat as inline shell script
	return exec.CommandContext(ctx, "sh", "-c", script)
}
