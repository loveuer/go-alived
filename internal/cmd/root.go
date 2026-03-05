package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Version can be set at build time via ldflags
var Version = "1.2.1"

var rootCmd = &cobra.Command{
	Use:   "go-alived",
	Short: "Go-Alived - VRRP High Availability Service",
	Long: `go-alived is a lightweight, dependency-free VRRP implementation in Go.
It provides high availability for IP addresses with health checking support.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}