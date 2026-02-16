package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRedisPathOverride(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("RDV_DB_DIR", tmp))
	defer func() { _ = os.Unsetenv("RDV_DB_DIR") }()

	require.Contains(t, redisPath(), tmp)
	require.Equal(t, filepath.Join(tmp, "redis.yaml"), redisPath())
}

func TestRedisExportVars(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Setenv("RDV_DB_DIR", tmp))
	defer func() { _ = os.Unsetenv("RDV_DB_DIR") }()

	t.Run("basic profile with password", func(t *testing.T) {
		cfg := redisConfig{Profiles: map[string]redisProfile{
			"dev": {Host: "localhost", Port: "6379", Password: "secret123", DB: "0", TLS: false},
		}}
		b, _ := yaml.Marshal(cfg)
		require.NoError(t, os.WriteFile(redisPath(), b, 0o600))

		vars, err := RedisExportVars("dev")
		require.NoError(t, err)
		require.Equal(t, "redis://:secret123@localhost:6379/0", vars["REDIS_URL"])
		require.Equal(t, "localhost", vars["REDIS_HOST"])
		require.Equal(t, "6379", vars["REDIS_PORT"])
		require.Equal(t, "0", vars["REDIS_DB"])
		require.Equal(t, "false", vars["REDIS_TLS"])
		require.Equal(t, "secret123", vars["REDIS_PASSWORD"])
	})

	t.Run("passwordless profile", func(t *testing.T) {
		cfg := redisConfig{Profiles: map[string]redisProfile{
			"local": {Host: "127.0.0.1", Port: "6379", Password: "", DB: "1", TLS: false},
		}}
		b, _ := yaml.Marshal(cfg)
		require.NoError(t, os.WriteFile(redisPath(), b, 0o600))

		vars, err := RedisExportVars("local")
		require.NoError(t, err)
		require.Equal(t, "redis://127.0.0.1:6379/1", vars["REDIS_URL"])
		require.Equal(t, "127.0.0.1", vars["REDIS_HOST"])
		require.Equal(t, "6379", vars["REDIS_PORT"])
		require.Equal(t, "1", vars["REDIS_DB"])
		require.Equal(t, "false", vars["REDIS_TLS"])
		_, hasPassword := vars["REDIS_PASSWORD"]
		require.False(t, hasPassword, "REDIS_PASSWORD should not be present when password is empty")
	})

	t.Run("TLS profile", func(t *testing.T) {
		cfg := redisConfig{Profiles: map[string]redisProfile{
			"prod": {Host: "redis.example.com", Port: "6380", Password: "prodpass", DB: "2", TLS: true},
		}}
		b, _ := yaml.Marshal(cfg)
		require.NoError(t, os.WriteFile(redisPath(), b, 0o600))

		vars, err := RedisExportVars("prod")
		require.NoError(t, err)
		require.Equal(t, "rediss://:prodpass@redis.example.com:6380/2", vars["REDIS_URL"])
		require.Equal(t, "redis.example.com", vars["REDIS_HOST"])
		require.Equal(t, "6380", vars["REDIS_PORT"])
		require.Equal(t, "2", vars["REDIS_DB"])
		require.Equal(t, "true", vars["REDIS_TLS"])
		require.Equal(t, "prodpass", vars["REDIS_PASSWORD"])
	})

	t.Run("profile not found", func(t *testing.T) {
		cfg := redisConfig{Profiles: map[string]redisProfile{}}
		b, _ := yaml.Marshal(cfg)
		require.NoError(t, os.WriteFile(redisPath(), b, 0o600))

		_, err := RedisExportVars("nonexistent")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}
