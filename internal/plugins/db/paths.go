package db

import (
	"os"
	"path/filepath"
)

// configDir returns ~/.config/rdv/db (or override via RDV_DB_DIR for tests)
func configDir() string {
	if v := os.Getenv("RDV_DB_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "rdv", "db")
}

func postgresPath() string {
	return filepath.Join(configDir(), "postgres.yaml")
}

func mysqlPath() string {
	return filepath.Join(configDir(), "mysql.yaml")
}

func redisPath() string {
	return filepath.Join(configDir(), "redis.yaml")
}
