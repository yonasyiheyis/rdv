package plugin

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// dummy plugin for testing
type testPlugin struct{}

func (t *testPlugin) Name() string { return "test" }
func (t *testPlugin) Register(root *cobra.Command) {
	root.AddCommand(&cobra.Command{Use: "test"})
}

func TestRegisterAndLoad(t *testing.T) {
	// fresh cmd
	root := &cobra.Command{Use: "rdv"}

	// register fake plugin
	Register(&testPlugin{})

	// load into cobra
	LoadAll(root)

	// ensure sub‑command exists
	cmd, _, err := root.Find([]string{"test"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
}
