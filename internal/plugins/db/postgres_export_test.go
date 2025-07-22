package db

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPgURLFormat(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("RDV_DB_DIR", tmp))

	cfg := pgConfig{Profiles: map[string]pgProfile{
		"ci": {Host: "localhost", Port: "5432", User: "bob", Password: "p@ss", DBName: "shop"},
	}}
	b, _ := yaml.Marshal(cfg)
	require.NoError(t, os.WriteFile(postgresPath(), b, 0o600))

	// capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	require.NoError(t, pgExport("ci"))

	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	require.Contains(t, string(out), "postgres://bob:p@ss@localhost:5432/shop?sslmode=disable")
}
