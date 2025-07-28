package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMySQLPathOverride(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("RDV_DB_DIR", tmp))
	defer func() { _ = os.Unsetenv("RDV_DB_DIR") }()

	require.Contains(t, mysqlPath(), tmp)
	require.Equal(t, filepath.Join(tmp, "mysql.yaml"), mysqlPath())
}
