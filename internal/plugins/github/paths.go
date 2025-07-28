package github

import (
	"os"
	"path/filepath"
)

func cfgPath() string {
	if v := os.Getenv("RDV_GH_DIR"); v != "" {
		return filepath.Join(v, "github.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "rdv", "github.yaml")
}
