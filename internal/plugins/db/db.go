package db

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonasyiheyis/rdv/internal/plugin"
)

type dbPlugin struct{}

func (d *dbPlugin) Name() string { return "db" }

func (d *dbPlugin) Register(root *cobra.Command) {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage DB connection settings (stub)",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("DB plugin is not implemented yet 🤓")
		},
	}
	root.AddCommand(dbCmd)
}

func init() {
	plugin.Register(&dbPlugin{})
}
