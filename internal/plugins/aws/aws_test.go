package aws

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathsOverride(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(tmp, "creds")))
	require.NoError(t, os.Setenv("AWS_CONFIG_FILE", filepath.Join(tmp, "config")))

	require.Contains(t, credentialsPath(), tmp)
	require.Contains(t, configPath(), tmp)
}
