package gcp

import (
	"os"
	"path/filepath"
)

// getConfigDir returns the GCP config directory path
func getConfigDir() string {
	if v := os.Getenv("RDV_GCP_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "rdv", "gcp")
}

// getConfigPath returns the full path to a GCP profile config file
func getConfigPath(profile string) string {
	return filepath.Join(getConfigDir(), profile+".yaml")
}

// getEnvPath returns the full path to a GCP profile env file
func getEnvPath(profile string) string {
	return filepath.Join(getConfigDir(), profile+".env")
}

// getCopiedKeyPath returns the full path to a copied GCP service account key file
func getCopiedKeyPath(profile string) string {
	return filepath.Join(getConfigDir(), profile+".json")
}
