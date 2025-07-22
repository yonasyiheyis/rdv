package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigDirOverride(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("RDV_DB_DIR", tmp))
	defer func() {
		require.NoError(t, os.Unsetenv("RDV_DB_DIR"))
	}()

	require.Contains(t, configDir(), tmp)
	require.Equal(t, filepath.Join(tmp, "postgres.yaml"), postgresPath())
}
