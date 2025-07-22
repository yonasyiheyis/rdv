package main

import (
	// Standard & third‑party
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	// Internal packages
	"github.com/yonasyiheyis/rdv/internal/logger"
	"github.com/yonasyiheyis/rdv/internal/plugin"
	"github.com/yonasyiheyis/rdv/internal/version"

	// --- side‑effect plugin imports ---
	_ "github.com/yonasyiheyis/rdv/internal/plugins/aws"
	_ "github.com/yonasyiheyis/rdv/internal/plugins/db"
)

var (
	cfgFile string
	debug   bool
	log     *zap.SugaredLogger
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

	/* ---------- initialise logger AFTER flags parsed ---------- */
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if log != nil { // already initialised (nested sub‑command)
			return nil
		}

		var zl *zap.Logger
		var err error
		if debug {
			zl, err = zap.NewDevelopment()
		} else {
			zl, err = zap.NewProduction()
		}
		if err != nil {
			return err
		}
		log = zl.Sugar()
		logger.L = log // make available to plugins
		return nil
	}

	// Pre‑run: init Viper and (future) logger
	cobra.OnInitialize(initConfig)

	// Placeholder for future sub‑commands; leaving root runnable alone.

	cmd.AddCommand(newCompletionCmd())

	// ----- Load plugin sub‑commands -----
	plugin.LoadAll(cmd)

	return cmd
}

func newCompletionCmd() *cobra.Command {
	comp := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion script",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Hidden:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				err = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				err = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return err
		},
	}
	return comp
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
		logger.L.Infof("using config file: %s", viper.ConfigFileUsed())
	}
}

// Execute is called by main.go.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
