package aws

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonasyiheyis/rdv/internal/plugin"
)

type awsPlugin struct{}

func (a *awsPlugin) Name() string { return "aws" }

func (a *awsPlugin) Register(root *cobra.Command) {
	awsCmd := &cobra.Command{
		Use:   "aws",
		Short: "Manage AWS credentials & profiles (stub)",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("AWS plugin is not implemented yet 🤓")
		},
	}
	root.AddCommand(awsCmd)
}

// Side‑effect import registration
func init() {
	plugin.Register(&awsPlugin{})
}
