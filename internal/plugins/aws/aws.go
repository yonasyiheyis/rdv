package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"

	"github.com/yonasyiheyis/rdv/internal/logger"
	"github.com/yonasyiheyis/rdv/internal/plugin"
	"github.com/yonasyiheyis/rdv/internal/ui"
)

// ---------- plugin wiring ----------

type awsPlugin struct{}

func (a *awsPlugin) Name() string { return "aws" }

func (a *awsPlugin) Register(root *cobra.Command) {
	awsCmd := &cobra.Command{
		Use:   "aws",
		Short: "Manage AWS credentials & profiles",
	}

	// -------- set-config ------------
	var setProfile string
	var setTestConn bool
	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set AWS credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSet(setProfile, setTestConn)
		},
	}
	setCmd.Flags().StringVarP(&setProfile, "profile", "p", "default", "AWS profile")
	setCmd.Flags().BoolVar(&setTestConn, "test-conn", false, "validate credentials with STS after saving")

	// -------- modify ----------------
	var modProfile string
	var modTestConn bool
	modifyCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing AWS profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runModifyAWS(modProfile, modTestConn)
		},
	}
	modifyCmd.Flags().StringVarP(&modProfile, "profile", "p", "default", "AWS profile")
	modifyCmd.Flags().BoolVar(&modTestConn, "test-conn", false, "validate credentials with STS after saving")

	// -------- delete ----------------
	var delProfile string
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an AWS profile from ~/.aws files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDeleteAWS(delProfile)
		},
	}
	deleteCmd.Flags().StringVarP(&delProfile, "profile", "p", "default", "AWS profile")

	// -------- export ----------------
	var expProfile string
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Print AWS_* export lines for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(expProfile)
		},
	}
	exportCmd.Flags().StringVarP(&expProfile, "profile", "p", "default", "AWS profile")

	awsCmd.AddCommand(setCmd, modifyCmd, deleteCmd, exportCmd)
	root.AddCommand(awsCmd)
}

func init() {
	plugin.Register(&awsPlugin{})
}

// ---------- data types ----------

type credsInput struct {
	AccessKey string
	SecretKey string
	Region    string
}

// ---------- helpers (load/save) ----------

func loadAWSProfile(profile string) (credsInput, error) {
	out := credsInput{}

	credINI, err := ini.Load(credentialsPath())
	if err != nil && !os.IsNotExist(err) {
		return out, err
	}
	if err == nil {
		sec := credINI.Section(profile)
		out.AccessKey = sec.Key("aws_access_key_id").String()
		out.SecretKey = sec.Key("aws_secret_access_key").String()
	}

	cfgINI, _ := ini.Load(configPath())
	if cfgINI != nil {
		sec := cfgINI.Section("profile " + profile)
		out.Region = sec.Key("region").String()
	}

	return out, nil
}

func saveAWSProfile(profile string, in credsInput) error {
	// ensure dir
	if err := os.MkdirAll(filepath.Dir(credentialsPath()), 0o700); err != nil {
		return err
	}

	// credentials
	credINI := ini.Empty()
	if fi, err := os.Stat(credentialsPath()); err == nil && fi.Size() > 0 {
		credINI, _ = ini.Load(credentialsPath())
	}
	csec := credINI.Section(profile)
	csec.Key("aws_access_key_id").SetValue(in.AccessKey)
	csec.Key("aws_secret_access_key").SetValue(in.SecretKey)
	if err := credINI.SaveTo(credentialsPath()); err != nil {
		return err
	}

	// config
	cfgINI := ini.Empty()
	if fi, err := os.Stat(configPath()); err == nil && fi.Size() > 0 {
		cfgINI, _ = ini.Load(configPath())
	}
	cfgSec := cfgINI.Section("profile " + profile)
	cfgSec.Key("region").SetValue(in.Region)
	return cfgINI.SaveTo(configPath())
}

// ---------- command impls ----------

func runSet(profile string, testConn bool) error {
	in := credsInput{}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("AWS Access Key ID").Value(&in.AccessKey).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("AWS Secret Access Key").
				EchoMode(huh.EchoModePassword).
				Value(&in.SecretKey).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("Default Region (e.g. us-east-1)").Value(&in.Region).Validate(huh.ValidateNotEmpty()),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	if err := saveAWSProfile(profile, in); err != nil {
		return err
	}

	logger.L.Infow("aws credentials saved", "profile", profile, "file", credentialsPath())
	fmt.Printf("✅ Credentials saved to %s (profile %q)\n", credentialsPath(), profile)

	if testConn {
		return testAWSCreds(in)
	}
	return nil
}

func runModifyAWS(profile string, testConn bool) error {
	current, err := loadAWSProfile(profile)
	if err != nil {
		return err
	}

	in := current // pre-fill
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("AWS Access Key ID").Value(&in.AccessKey).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("AWS Secret Access Key").
				EchoMode(huh.EchoModePassword).
				Value(&in.SecretKey).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("Default Region").Value(&in.Region).Validate(huh.ValidateNotEmpty()),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	if err := saveAWSProfile(profile, in); err != nil {
		return err
	}

	logger.L.Infow("aws profile modified", "profile", profile)
	fmt.Printf("✅ Updated profile %q\n", profile)

	if testConn {
		return testAWSCreds(in)
	}
	return nil
}

func runDeleteAWS(profile string) error {
	ok, err := ui.Confirm(fmt.Sprintf("Delete AWS profile %q?", profile))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Aborted.")
		return nil
	}

	credINI, _ := ini.Load(credentialsPath())
	cfgINI, _ := ini.Load(configPath())

	if credINI != nil {
		credINI.DeleteSection(profile)
		_ = credINI.SaveTo(credentialsPath())
	}
	if cfgINI != nil {
		cfgINI.DeleteSection("profile " + profile)
		_ = cfgINI.SaveTo(configPath())
	}

	logger.L.Infow("aws profile deleted", "profile", profile)
	fmt.Printf("🗑️  Deleted profile %q\n", profile)
	return nil
}

func runExport(profile string) error {
	credINI, err := ini.Load(credentialsPath())
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}
	sec := credINI.Section(profile)
	id := strings.TrimSpace(sec.Key("aws_access_key_id").String())
	secret := strings.TrimSpace(sec.Key("aws_secret_access_key").String())
	if id == "" || secret == "" {
		return fmt.Errorf("profile %q not found in %s", profile, credentialsPath())
	}

	cfgINI, _ := ini.Load(configPath())
	region := strings.TrimSpace(cfgINI.Section("profile " + profile).Key("region").String())

	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", id)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", secret)
	if region != "" {
		fmt.Printf("export AWS_DEFAULT_REGION=%s\n", region)
	}
	return nil
}
