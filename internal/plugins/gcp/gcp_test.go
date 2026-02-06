package gcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaths(t *testing.T) {
	home, _ := os.UserHomeDir()
	expectedDir := filepath.Join(home, ".config", "rdv", "gcp")

	require.Equal(t, expectedDir, getConfigDir())
	require.Equal(t, filepath.Join(expectedDir, "dev.yaml"), getConfigPath("dev"))
	require.Equal(t, filepath.Join(expectedDir, "dev.env"), getEnvPath("dev"))
	require.Equal(t, filepath.Join(expectedDir, "dev.json"), getCopiedKeyPath("dev"))
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
			hasError: false,
		},
		{
			name:     "relative path",
			input:    "test.json",
			expected: "", // will be different based on current working directory
			hasError: false,
		},
		{
			name:     "absolute path",
			input:    "/tmp/test.json",
			expected: "/tmp/test.json",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizePath(tt.input)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				switch tt.input {
				case "":
					require.Equal(t, "", result)
				case "/tmp/test.json":
					require.Equal(t, "/tmp/test.json", result)
				default:
					// For relative paths, can't predict the exact result
				}
			}
		})
	}
}

func TestGenerateEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		config   gcpConfig
		expected map[string]string
	}{
		{
			name: "service account with all fields",
			config: gcpConfig{
				Auth:          "service-account-json",
				KeyFile:       "/path/to/key.json",
				CopiedKeyFile: "/copied/key.json",
				ProjectID:     "test-project",
				Region:        "us-central1",
				Zone:          "us-central1-a",
			},
			expected: map[string]string{
				"GOOGLE_APPLICATION_CREDENTIALS": "/copied/key.json",
				"CLOUDSDK_CORE_PROJECT":          "test-project",
				"GOOGLE_CLOUD_PROJECT":           "test-project",
				"GOOGLE_CLOUD_REGION":            "us-central1",
				"GOOGLE_CLOUD_ZONE":              "us-central1-a",
			},
		},
		{
			name: "gcloud-adc with minimal fields",
			config: gcpConfig{
				Auth:      "gcloud-adc",
				ProjectID: "test-project",
			},
			expected: map[string]string{
				"CLOUDSDK_CORE_PROJECT": "test-project",
				"GOOGLE_CLOUD_PROJECT":  "test-project",
			},
		},
		{
			name: "service account without copied key",
			config: gcpConfig{
				Auth:      "service-account-json",
				KeyFile:   "/path/to/key.json",
				ProjectID: "test-project",
			},
			expected: map[string]string{
				"GOOGLE_APPLICATION_CREDENTIALS": "/path/to/key.json",
				"CLOUDSDK_CORE_PROJECT":          "test-project",
				"GOOGLE_CLOUD_PROJECT":           "test-project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateEnvVars(tt.config)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRunList(t *testing.T) {
	// Test with no profiles (when config directory doesn't exist)
	t.Run("no profiles", func(t *testing.T) {
		// This will work because the config directory likely doesn't exist in test environment
		err := runList()
		require.NoError(t, err)
	})
}
