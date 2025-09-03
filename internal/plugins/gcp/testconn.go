package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ServiceAccountKey represents the structure of a GCP service account JSON key file
type ServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func testGCPConnection(config gcpConfig) error {
	switch config.Auth {
	case "service-account-json":
		return testServiceAccountAuth(config)
	case "gcloud-adc":
		return testGcloudADC()
	default:
		return fmt.Errorf("unsupported authentication method: %s", config.Auth)
	}
}

func testServiceAccountAuth(config gcpConfig) error {
	// Determine which key file to use
	keyPath := config.KeyFile
	if config.CopiedKeyFile != "" {
		keyPath = config.CopiedKeyFile
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("key file does not exist: %s", keyPath)
	}

	// Read and parse the key file
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	var key ServiceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return fmt.Errorf("failed to parse key file as JSON: %w", err)
	}

	// Validate required fields
	if key.Type == "" {
		return fmt.Errorf("missing 'type' field in key file")
	}
	if key.ProjectID == "" {
		return fmt.Errorf("missing 'project_id' field in key file")
	}
	if key.PrivateKey == "" {
		return fmt.Errorf("missing 'private_key' field in key file")
	}
	if key.ClientEmail == "" {
		return fmt.Errorf("missing 'client_email' field in key file")
	}

	// Test with gcloud auth activate-service-account
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a temporary directory for the key file
	tmpDir, err := os.MkdirTemp("", "rdv-gcp-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			// Log error but don't fail the test
			fmt.Printf("Warning: failed to clean up temp directory %s: %v\n", tmpDir, err)
		}
	}()

	tmpKeyPath := filepath.Join(tmpDir, "key.json")
	if err := os.WriteFile(tmpKeyPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp key file: %w", err)
	}

	// Test authentication
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "activate-service-account", "--key-file", tmpKeyPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gcloud auth activate-service-account failed: %w", err)
	}

	// Test getting access token
	cmd = exec.CommandContext(ctx, "gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("gcloud auth print-access-token failed: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("gcloud auth print-access-token returned empty token")
	}

	fmt.Println("✅ GCP service account authentication is valid")
	return nil
}

func testGcloudADC() error {
	// Check if gcloud is installed
	if _, err := exec.LookPath("gcloud"); err != nil {
		return fmt.Errorf("gcloud CLI is not installed or not in PATH")
	}

	// Test getting access token with application default credentials
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("gcloud auth application-default print-access-token failed: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("gcloud auth application-default print-access-token returned empty token")
	}

	fmt.Println("✅ GCP application default credentials are valid")
	return nil
}
