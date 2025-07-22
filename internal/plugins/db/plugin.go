package db

import (
	"github.com/spf13/cobra"

	"github.com/yonasyiheyis/rdv/internal/plugin"
)

type dbPlugin struct{}

func (d *dbPlugin) Name() string { return "db" }

func (d *dbPlugin) Register(root *cobra.Command) {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage database connection settings",
	}

	// attach the postgres sub‑command
	dbCmd.AddCommand(newPostgresCmd())
	root.AddCommand(dbCmd)
}

func init() { plugin.Register(&dbPlugin{}) }
