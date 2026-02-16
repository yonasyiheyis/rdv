package gcp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExportVars_ServiceAccount(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("RDV_GCP_DIR", tmp)

	profile := "test"
	cfg := gcpConfig{
		Auth:      "service-account-json",
		ProjectID: "my-project",
		Region:    "us-central1",
		Zone:      "us-central1-a",
		KeyFile:   "/path/to/key.json",
	}

	// Write config file to temp dir
	configPath := getConfigPath(profile)
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0o600))

	vars, err := ExportVars(profile)
	require.NoError(t, err)
	require.Equal(t, "my-project", vars["CLOUDSDK_CORE_PROJECT"])
	require.Equal(t, "my-project", vars["GOOGLE_CLOUD_PROJECT"])
	require.Equal(t, "us-central1", vars["GOOGLE_CLOUD_REGION"])
	require.Equal(t, "us-central1-a", vars["GOOGLE_CLOUD_ZONE"])
	require.Equal(t, "/path/to/key.json", vars["GOOGLE_APPLICATION_CREDENTIALS"])
}

func TestExportVars_ADC(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("RDV_GCP_DIR", tmp)

	profile := "test-adc"
	cfg := gcpConfig{
		Auth:      "gcloud-adc",
		ProjectID: "my-adc-project",
		Region:    "europe-west1",
	}

	// Write config file
	configPath := getConfigPath(profile)
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0o600))

	vars, err := ExportVars(profile)
	require.NoError(t, err)
	require.Equal(t, "my-adc-project", vars["CLOUDSDK_CORE_PROJECT"])
	require.Equal(t, "my-adc-project", vars["GOOGLE_CLOUD_PROJECT"])
	require.Equal(t, "europe-west1", vars["GOOGLE_CLOUD_REGION"])
	// Should NOT set GOOGLE_APPLICATION_CREDENTIALS for ADC
	_, hasCredentials := vars["GOOGLE_APPLICATION_CREDENTIALS"]
	require.False(t, hasCredentials)
}

func TestExportVars_ProfileNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("RDV_GCP_DIR", tmp)

	_, err := ExportVars("nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile \"nonexistent\" not found")
}

func TestExportVars_EmptyAuth(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("RDV_GCP_DIR", tmp)

	profile := "empty"
	cfg := gcpConfig{
		Auth:      "", // Empty auth field
		ProjectID: "my-project",
	}

	// Write config file
	configPath := getConfigPath(profile)
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0o600))

	_, err = ExportVars(profile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile \"empty\" not found")
}
