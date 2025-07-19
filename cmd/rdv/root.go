package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yonasyiheyis/rdv/internal/version"
)

var (
	cfgFile string
	debug   bool
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rdv",
		Short: "ReadyDev (rdv) – interactive dev‑env config manager",
		Long: `rdv (readyDev) is a plugin‑based CLI that sets, modifies,
deletes, and exports configuration for AWS, databases, GitHub, and more.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}

	// Global flags
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: $HOME/.config/rdv/rdv.yaml)")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// Built‑in Cobra version flag override to show full details
	cmd.SetVersionTemplate(fmt.Sprintf("rdv %s (commit: %s, date: %s)\n",
		version.Version, version.Commit, version.Date))

	// Pre‑run: init Viper and (future) logger
	cobra.OnInitialize(initConfig)

	// Placeholder for future sub‑commands; leaving root runnable alone.

	return cmd
}

// initConfig wires Viper to read config + env vars.
func initConfig() {
	// 1. Determine config path
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, _ := os.UserConfigDir() // ~/.config
		viper.AddConfigPath(filepath.Join(configDir, "rdv"))
		viper.AddConfigPath("/etc/rdv/")
		viper.SetConfigName("rdv") // rdv.{yaml|yml|json|toml}
	}

	// 2. ENV support (e.g., RDV_AWS_REGION)
	viper.SetEnvPrefix("RDV")
	viper.AutomaticEnv()

	// 3. Read file if present; ignore if not found
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintf(os.Stderr, "using config file: %s\n", viper.ConfigFileUsed())
	}
}

// Execute is called by main.go.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
