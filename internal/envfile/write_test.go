package envfile

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteEnv(t *testing.T) {
	tmp := t.TempDir() + "/.env"
	require.NoError(t, WriteEnv(tmp, map[string]string{"USER": "test-user"}))
	require.NoError(t, WriteEnv(tmp, map[string]string{"TOKEN": "test-token"}))

	b, _ := os.ReadFile(tmp)
	require.Contains(t, string(b), "USER=test-user")
	require.Contains(t, string(b), "TOKEN=test-token")
}
