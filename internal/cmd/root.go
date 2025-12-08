package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-alived",
	Short: "Go-Alived - VRRP High Availability Service",
	Long: `go-alived is a lightweight, dependency-free VRRP implementation in Go.
It provides high availability for IP addresses with health checking support.`,
	Version: "1.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}