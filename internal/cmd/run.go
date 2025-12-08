package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/loveuer/go-alived/internal/health"
	"github.com/loveuer/go-alived/internal/vrrp"
	"github.com/loveuer/go-alived/pkg/config"
	"github.com/loveuer/go-alived/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	configFile string
	debug      bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the VRRP service",
	Long:  `Start the go-alived VRRP service with health checking.`,
	Run:   runService,
}

func init() {
	rootCmd.AddCommand(runCmd)
	
	runCmd.Flags().StringVarP(&configFile, "config", "c", "/etc/go-alived/config.yaml", "path to configuration file")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")
}

func runService(cmd *cobra.Command, args []string) {
	log := logger.New(debug)

	log.Info("starting go-alived...")
	log.Info("loading configuration from: %s", configFile)

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Error("failed to load configuration: %v", err)
		os.Exit(1)
	}

	log.Info("configuration loaded successfully")
	log.Debug("config: %+v", cfg)

	healthMgr, err := health.LoadFromConfig(cfg, log)
	if err != nil {
		log.Error("failed to load health check configuration: %v", err)
		os.Exit(1)
	}

	vrrpMgr := vrrp.NewManager(log)
	if err := vrrpMgr.LoadFromConfig(cfg); err != nil {
		log.Error("failed to load VRRP configuration: %v", err)
		os.Exit(1)
	}

	setupHealthTracking(vrrpMgr, healthMgr, log)

	healthMgr.StartAll()

	if err := vrrpMgr.StartAll(); err != nil {
		log.Error("failed to start VRRP instances: %v", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			log.Info("received SIGHUP, reloading configuration...")
			newCfg, err := config.Load(configFile)
			if err != nil {
				log.Error("failed to reload configuration: %v", err)
				continue
			}
			if err := vrrpMgr.Reload(newCfg); err != nil {
				log.Error("failed to reload VRRP: %v", err)
				continue
			}
			cfg = newCfg
			log.Info("configuration reloaded successfully")
		case syscall.SIGINT, syscall.SIGTERM:
			log.Info("received signal %v, shutting down...", sig)
			cleanup(log, vrrpMgr, healthMgr)
			os.Exit(0)
		}
	}
}

func cleanup(log *logger.Logger, vrrpMgr *vrrp.Manager, healthMgr *health.Manager) {
	log.Info("cleaning up resources...")
	healthMgr.StopAll()
	vrrpMgr.StopAll()
}

func setupHealthTracking(vrrpMgr *vrrp.Manager, healthMgr *health.Manager, log *logger.Logger) {
	instances := vrrpMgr.GetAllInstances()

	for _, inst := range instances {
		for _, trackScript := range inst.TrackScripts {
			monitor, ok := healthMgr.GetMonitor(trackScript)
			if !ok {
				log.Warn("[%s] track_script '%s' not found in health checkers", inst.Name, trackScript)
				continue
			}

			instanceName := inst.Name
			monitor.OnStateChange(func(checkerName string, oldHealthy, newHealthy bool) {
				vrrpInst, ok := vrrpMgr.GetInstance(instanceName)
				if !ok {
					return
				}

				if newHealthy && !oldHealthy {
					log.Info("[%s] health check '%s' recovered, resetting priority", instanceName, checkerName)
					vrrpInst.ResetPriority()
				} else if !newHealthy && oldHealthy {
					log.Warn("[%s] health check '%s' failed, decreasing priority", instanceName, checkerName)
					vrrpInst.AdjustPriority(-10)
				}
			})

			log.Info("[%s] tracking health check: %s", inst.Name, trackScript)
		}
	}
}
