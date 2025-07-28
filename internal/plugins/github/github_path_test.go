package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCfgPathOverride(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("RDV_GH_DIR", tmp))
	defer func() { _ = os.Unsetenv("RDV_GH_DIR") }()

	require.Contains(t, cfgPath(), tmp)
	require.Equal(t, filepath.Join(tmp, "github.yaml"), cfgPath())
}
