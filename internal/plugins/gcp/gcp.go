package gcp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yonasyiheyis/rdv/internal/cli"
	"github.com/yonasyiheyis/rdv/internal/envfile"
	"github.com/yonasyiheyis/rdv/internal/exitcodes"
	fflags "github.com/yonasyiheyis/rdv/internal/flags"
	"github.com/yonasyiheyis/rdv/internal/logger"
	"github.com/yonasyiheyis/rdv/internal/plugin"
	iprint "github.com/yonasyiheyis/rdv/internal/print"
	"github.com/yonasyiheyis/rdv/internal/ui"
)

// ---------- plugin wiring ----------

type gcpPlugin struct{}

func (g *gcpPlugin) Name() string { return "gcp" }

func (g *gcpPlugin) Register(root *cobra.Command) {
	gcpCmd := &cobra.Command{
		Use:   "gcp",
		Short: "Manage GCP credentials & profiles",
	}

	// -------- set-config ------------
	var setProfile string
	var setAuth string
	var setProjectID string
	var setRegion string
	var setZone string
	var setKeyFile string
	var setCopyKey bool
	var setTestConn bool
	var setNoPrompt bool

	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Set GCP configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetConfig(setProfile, setAuth, setProjectID, setRegion, setZone, setKeyFile, setCopyKey, setTestConn, setNoPrompt)
		},
	}
	fflags.AddNoPromptFlag(setCmd.Flags(), &setNoPrompt)
	setCmd.Flags().StringVarP(&setProfile, "profile", "p", "dev", "GCP profile")
	setCmd.Flags().StringVar(&setAuth, "auth", "", "Authentication method: service-account-json or gcloud-adc")
	setCmd.Flags().StringVar(&setProjectID, "project-id", "", "GCP project ID")
	setCmd.Flags().StringVar(&setRegion, "region", "", "GCP region (e.g. us-central1)")
	setCmd.Flags().StringVar(&setZone, "zone", "", "GCP zone (e.g. us-central1-a)")
	setCmd.Flags().StringVar(&setKeyFile, "key-file", "", "Path to service account JSON key file")
	setCmd.Flags().BoolVar(&setCopyKey, "copy-key", false, "Copy key file to config directory")
	setCmd.Flags().BoolVar(&setTestConn, "test-conn", false, "Test connection after saving")

	// -------- modify ----------------
	var modProfile string
	var modAuth string
	var modProjectID string
	var modRegion string
	var modZone string
	var modKeyFile string
	var modCopyKey bool
	var modTestConn bool
	var modNoPrompt bool

	modifyCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing GCP profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runModify(modProfile, modAuth, modProjectID, modRegion, modZone, modKeyFile, modCopyKey, modTestConn, modNoPrompt)
		},
	}
	fflags.AddNoPromptFlag(modifyCmd.Flags(), &modNoPrompt)
	modifyCmd.Flags().StringVarP(&modProfile, "profile", "p", "dev", "GCP profile")
	modifyCmd.Flags().StringVar(&modAuth, "auth", "", "Authentication method: service-account-json or gcloud-adc")
	modifyCmd.Flags().StringVar(&modProjectID, "project-id", "", "GCP project ID")
	modifyCmd.Flags().StringVar(&modRegion, "region", "", "GCP region")
	modifyCmd.Flags().StringVar(&modZone, "zone", "", "GCP zone")
	modifyCmd.Flags().StringVar(&modKeyFile, "key-file", "", "Path to service account JSON key file")
	modifyCmd.Flags().BoolVar(&modCopyKey, "copy-key", false, "Copy key file to config directory")
	modifyCmd.Flags().BoolVar(&modTestConn, "test-conn", false, "Test connection after saving")

	// -------- delete ----------------
	var delProfile string
	var delPurgeKey bool

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GCP profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDelete(delProfile, delPurgeKey)
		},
	}
	deleteCmd.Flags().StringVarP(&delProfile, "profile", "p", "dev", "GCP profile")
	deleteCmd.Flags().BoolVar(&delPurgeKey, "purge-key", false, "Delete copied key file if present")

	// -------- list ----------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List GCP profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList()
		},
	}

	// -------- show ----------------
	var showProfile string

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show GCP profile configuration (sanitized)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runShow(showProfile)
		},
	}
	showCmd.Flags().StringVarP(&showProfile, "profile", "p", "dev", "GCP profile")

	// -------- export ----------------
	var expProfile string
	var expPrint bool
	var expStyle string

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export GCP environment variables",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(expProfile, expPrint, expStyle)
		},
	}
	exportCmd.Flags().StringVarP(&expProfile, "profile", "p", "dev", "GCP profile")
	exportCmd.Flags().BoolVar(&expPrint, "print", false, "Print environment variables to stdout")
	exportCmd.Flags().StringVar(&expStyle, "style", "dotenv", "Output style: dotenv or export")

	// -------- test-conn ----------------
	var testProfile string

	testConnCmd := &cobra.Command{
		Use:   "test-conn",
		Short: "Test GCP connection",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTestConn(testProfile)
		},
	}
	testConnCmd.Flags().StringVarP(&testProfile, "profile", "p", "dev", "GCP profile")

	gcpCmd.AddCommand(setCmd, modifyCmd, deleteCmd, listCmd, showCmd, exportCmd, testConnCmd)
	root.AddCommand(gcpCmd)
}

func init() {
	plugin.Register(&gcpPlugin{})
}

// ---------- data types ----------

type gcpConfig struct {
	Auth          string    `yaml:"auth"` // service-account-json or gcloud-adc
	KeyFile       string    `yaml:"key_file,omitempty"`
	CopiedKeyFile string    `yaml:"copied_key_file,omitempty"`
	ProjectID     string    `yaml:"project_id"`
	Region        string    `yaml:"region,omitempty"`
	Zone          string    `yaml:"zone,omitempty"`
	UpdatedAt     time.Time `yaml:"updated_at"`
}

type gcpConfigInput struct {
	Auth      string
	KeyFile   string
	ProjectID string
	Region    string
	Zone      string
	CopyKey   bool
}

// ---------- helpers (load/save) ----------

func loadGCPConfig(profile string) (gcpConfig, error) {
	var config gcpConfig
	configPath := getConfigPath(profile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // return empty config if file doesn't exist
		}
		return config, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func saveGCPConfig(profile string, config gcpConfig) error {
	configPath := getConfigPath(profile)
	configDir := filepath.Dir(configPath)

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	// Update timestamp
	config.UpdatedAt = time.Now()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// normalizePath converts relative paths and ~ to absolute paths
func normalizePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Handle ~
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

// ---------- command implementations ----------

func runSetConfig(profile, auth, projectID, region, zone, keyFile string, copyKey, testConn, noPrompt bool) error {
	var input gcpConfigInput

	if noPrompt || !cli.IsInteractive() {
		// Non-interactive mode - validate required fields
		if auth == "" {
			return exitcodes.New(exitcodes.InvalidArgs, "missing required flag: --auth")
		}
		if auth != "service-account-json" && auth != "gcloud-adc" {
			return exitcodes.New(exitcodes.InvalidArgs, "auth must be 'service-account-json' or 'gcloud-adc'")
		}
		if projectID == "" {
			return exitcodes.New(exitcodes.InvalidArgs, "missing required flag: --project-id")
		}
		if auth == "service-account-json" && keyFile == "" {
			return exitcodes.New(exitcodes.InvalidArgs, "missing required flag: --key-file for service-account-json auth")
		}

		input = gcpConfigInput{
			Auth:      auth,
			KeyFile:   keyFile,
			ProjectID: projectID,
			Region:    region,
			Zone:      zone,
			CopyKey:   copyKey,
		}
	} else {
		// Interactive mode
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Authentication Method").
					Options(
						huh.NewOption("Service Account JSON", "service-account-json"),
						huh.NewOption("gcloud Application Default Credentials", "gcloud-adc"),
					).
					Value(&input.Auth).
					Validate(huh.ValidateNotEmpty()),
				huh.NewInput().
					Title("GCP Project ID").
					Value(&input.ProjectID).
					Validate(huh.ValidateNotEmpty()),
				huh.NewInput().
					Title("GCP Region (optional, e.g. us-central1)").
					Value(&input.Region),
				huh.NewInput().
					Title("GCP Zone (optional, e.g. us-central1-a)").
					Value(&input.Zone),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		// Additional form for service account JSON
		if input.Auth == "service-account-json" {
			keyForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Path to Service Account JSON Key File").
						Value(&input.KeyFile).
						Validate(huh.ValidateNotEmpty()),
					huh.NewConfirm().
						Title("Copy key file to config directory?").
						Value(&input.CopyKey),
				),
			)

			if err := keyForm.Run(); err != nil {
				return err
			}
		}
	}

	// Normalize paths
	normalizedKeyFile, err := normalizePath(input.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to normalize key file path: %w", err)
	}
	input.KeyFile = normalizedKeyFile

	// Create config
	config := gcpConfig{
		Auth:      input.Auth,
		KeyFile:   input.KeyFile,
		ProjectID: input.ProjectID,
		Region:    input.Region,
		Zone:      input.Zone,
	}

	// Handle key file copying
	if input.Auth == "service-account-json" && input.CopyKey {
		copiedPath := getCopiedKeyPath(profile)
		if err := copyKeyFile(input.KeyFile, copiedPath); err != nil {
			return fmt.Errorf("failed to copy key file: %w", err)
		}
		config.CopiedKeyFile = copiedPath
	}

	// Save config
	if err := saveGCPConfig(profile, config); err != nil {
		return err
	}

	logger.L.Infow("gcp config saved", "profile", profile, "file", getConfigPath(profile))
	fmt.Printf("✅ GCP configuration saved for profile %q\n", profile)

	// Test connection if requested
	if testConn {
		if err := testGCPConnection(config); err != nil {
			return exitcodes.Wrap(exitcodes.ConnectionFailed, err)
		}
	}

	return nil
}

func runModify(profile, auth, projectID, region, zone, keyFile string, copyKey, testConn, noPrompt bool) error {
	// Load existing config
	current, err := loadGCPConfig(profile)
	if err != nil {
		return err
	}

	// Check if profile exists
	if current.Auth == "" {
		return exitcodes.New(exitcodes.ProfileNotFound, fmt.Sprintf("profile %q not found", profile))
	}

	input := gcpConfigInput{
		Auth:      current.Auth,
		KeyFile:   current.KeyFile,
		ProjectID: current.ProjectID,
		Region:    current.Region,
		Zone:      current.Zone,
		CopyKey:   copyKey,
	}

	if noPrompt || !cli.IsInteractive() {
		// Non-interactive mode - only update provided fields
		if auth != "" {
			if auth != "service-account-json" && auth != "gcloud-adc" {
				return exitcodes.New(exitcodes.InvalidArgs, "auth must be 'service-account-json' or 'gcloud-adc'")
			}
			input.Auth = auth
		}
		if projectID != "" {
			input.ProjectID = projectID
		}
		if region != "" {
			input.Region = region
		}
		if zone != "" {
			input.Zone = zone
		}
		if keyFile != "" {
			input.KeyFile = keyFile
		}
	} else {
		// Interactive mode - show current values
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Authentication Method").
					Options(
						huh.NewOption("Service Account JSON", "service-account-json"),
						huh.NewOption("gcloud Application Default Credentials", "gcloud-adc"),
					).
					Value(&input.Auth).
					Validate(huh.ValidateNotEmpty()),
				huh.NewInput().
					Title("GCP Project ID").
					Value(&input.ProjectID).
					Validate(huh.ValidateNotEmpty()),
				huh.NewInput().
					Title("GCP Region (optional)").
					Value(&input.Region),
				huh.NewInput().
					Title("GCP Zone (optional)").
					Value(&input.Zone),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		// Additional form for service account JSON
		if input.Auth == "service-account-json" {
			keyForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Path to Service Account JSON Key File").
						Value(&input.KeyFile).
						Validate(huh.ValidateNotEmpty()),
					huh.NewConfirm().
						Title("Copy key file to config directory?").
						Value(&input.CopyKey),
				),
			)

			if err := keyForm.Run(); err != nil {
				return err
			}
		}
	}

	// Normalize paths
	normalizedKeyFile, err := normalizePath(input.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to normalize key file path: %w", err)
	}
	input.KeyFile = normalizedKeyFile

	// Create updated config
	config := gcpConfig{
		Auth:          input.Auth,
		KeyFile:       input.KeyFile,
		CopiedKeyFile: current.CopiedKeyFile, // preserve existing copied key file
		ProjectID:     input.ProjectID,
		Region:        input.Region,
		Zone:          input.Zone,
	}

	// Handle key file copying
	if input.Auth == "service-account-json" && input.CopyKey {
		copiedPath := getCopiedKeyPath(profile)
		if err := copyKeyFile(input.KeyFile, copiedPath); err != nil {
			return fmt.Errorf("failed to copy key file: %w", err)
		}
		config.CopiedKeyFile = copiedPath
	}

	// Save config
	if err := saveGCPConfig(profile, config); err != nil {
		return err
	}

	logger.L.Infow("gcp profile modified", "profile", profile)
	fmt.Printf("✅ Updated profile %q\n", profile)

	// Test connection if requested
	if testConn {
		if err := testGCPConnection(config); err != nil {
			return exitcodes.Wrap(exitcodes.ConnectionFailed, err)
		}
	}

	return nil
}

func runDelete(profile string, purgeKey bool) error {
	config, err := loadGCPConfig(profile)
	if err != nil {
		return err
	}

	if config.Auth == "" {
		return exitcodes.New(exitcodes.ProfileNotFound, fmt.Sprintf("profile %q not found", profile))
	}

	ok, err := ui.Confirm(fmt.Sprintf("Delete GCP profile %q?", profile))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Aborted.")
		return nil
	}

	// Delete config file
	configPath := getConfigPath(profile)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Delete env file
	envPath := getEnvPath(profile)
	if err := os.Remove(envPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Delete copied key file if requested and exists
	if purgeKey && config.CopiedKeyFile != "" {
		if err := os.Remove(config.CopiedKeyFile); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	logger.L.Infow("gcp profile deleted", "profile", profile)
	fmt.Printf("🗑️  Deleted profile %q\n", profile)
	return nil
}

func runShow(profile string) error {
	config, err := loadGCPConfig(profile)
	if err != nil {
		return err
	}

	if config.Auth == "" {
		return exitcodes.New(exitcodes.ProfileNotFound, fmt.Sprintf("profile %q not found", profile))
	}

	// Create sanitized version for display
	sanitized := gcpConfig{
		Auth:          config.Auth,
		KeyFile:       iprint.Redact(config.KeyFile),
		CopiedKeyFile: iprint.Redact(config.CopiedKeyFile),
		ProjectID:     config.ProjectID,
		Region:        config.Region,
		Zone:          config.Zone,
		UpdatedAt:     config.UpdatedAt,
	}

	if iprint.JSON {
		return iprint.Out(map[string]any{
			"profile": profile,
			"config":  sanitized,
		})
	}

	fmt.Printf("profile: %s\n", profile)
	fmt.Printf("  auth: %s\n", sanitized.Auth)
	if sanitized.KeyFile != "" {
		fmt.Printf("  key_file: %s\n", sanitized.KeyFile)
	}
	if sanitized.CopiedKeyFile != "" {
		fmt.Printf("  copied_key_file: %s\n", sanitized.CopiedKeyFile)
	}
	fmt.Printf("  project_id: %s\n", sanitized.ProjectID)
	if sanitized.Region != "" {
		fmt.Printf("  region: %s\n", sanitized.Region)
	}
	if sanitized.Zone != "" {
		fmt.Printf("  zone: %s\n", sanitized.Zone)
	}
	fmt.Printf("  updated_at: %s\n", sanitized.UpdatedAt.Format(time.RFC3339))

	return nil
}

func runExport(profile string, print bool, style string) error {
	config, err := loadGCPConfig(profile)
	if err != nil {
		return err
	}

	if config.Auth == "" {
		return exitcodes.New(exitcodes.ProfileNotFound, fmt.Sprintf("profile %q not found", profile))
	}

	// Generate environment variables
	vars, err := generateEnvVars(config)
	if err != nil {
		return err
	}

	if print {
		// Print to stdout
		if style == "export" {
			for k, v := range vars {
				fmt.Printf("export %s=\"%s\"\n", k, v)
			}
		} else {
			for k, v := range vars {
				fmt.Printf("%s=%s\n", k, v)
			}
		}
		return nil
	}

	// Write to .env file
	envPath := getEnvPath(profile)
	if err := envfile.WriteEnv(envPath, vars); err != nil {
		return exitcodes.Wrap(exitcodes.EnvWriteFailed, err)
	}

	if iprint.JSON {
		return iprint.Out(map[string]any{
			"path": envPath,
			"vars": vars,
		})
	}

	fmt.Printf("✅ wrote %d vars to %s\n", len(vars), envPath)
	return nil
}

func runTestConn(profile string) error {
	config, err := loadGCPConfig(profile)
	if err != nil {
		return err
	}

	if config.Auth == "" {
		return exitcodes.New(exitcodes.ProfileNotFound, fmt.Sprintf("profile %q not found", profile))
	}

	return testGCPConnection(config)
}

func runList() error {
	configDir := getConfigDir()

	// Check if config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if iprint.JSON {
			return iprint.Out(map[string]any{"profiles": []string{}})
		}
		fmt.Println("(no profiles)")
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config directory: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only include .yaml files and extract profile name
		if strings.HasSuffix(name, ".yaml") {
			profile := strings.TrimSuffix(name, ".yaml")
			profiles = append(profiles, profile)
		}
	}

	// Sort profiles for consistent output
	sort.Strings(profiles)

	if iprint.JSON {
		return iprint.Out(map[string]any{"profiles": profiles})
	}

	if len(profiles) == 0 {
		fmt.Println("(no profiles)")
		return nil
	}

	for _, profile := range profiles {
		fmt.Println(profile)
	}

	return nil
}

// ---------- helper functions ----------

func copyKeyFile(src, dst string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return err
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write to destination
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return err
	}

	return nil
}

func generateEnvVars(config gcpConfig) (map[string]string, error) {
	vars := map[string]string{
		"CLOUDSDK_CORE_PROJECT": config.ProjectID,
		"GOOGLE_CLOUD_PROJECT":  config.ProjectID,
	}

	if config.Region != "" {
		vars["GOOGLE_CLOUD_REGION"] = config.Region
	}

	if config.Zone != "" {
		vars["GOOGLE_CLOUD_ZONE"] = config.Zone
	}

	// For service account JSON, set GOOGLE_APPLICATION_CREDENTIALS
	if config.Auth == "service-account-json" {
		keyPath := config.KeyFile
		if config.CopiedKeyFile != "" {
			keyPath = config.CopiedKeyFile
		}
		vars["GOOGLE_APPLICATION_CREDENTIALS"] = keyPath
	}

	return vars, nil
}
