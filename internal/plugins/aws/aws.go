package aws

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"

	"github.com/yonasyiheyis/rdv/internal/plugin"
)

type awsPlugin struct{}

func (a *awsPlugin) Name() string { return "aws" }

func (a *awsPlugin) Register(root *cobra.Command) {
	awsCmd := &cobra.Command{
		Use:   "aws",
		Short: "Manage AWS credentials & profiles",
	}

	// -------- set-config ------------
	var profile string
	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set AWS credentials",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runSet(profile) },
	}
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile")

	// -------- export ----------------
	var expProfile string
	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print AWS_* export lines for a profile",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runExport(expProfile) },
	}
	expCmd.Flags().StringVarP(&expProfile, "profile", "p", "default", "AWS profile")

	awsCmd.AddCommand(setCmd, expCmd)
	root.AddCommand(awsCmd)
}

// Side‑effect import registration
func init() {
	plugin.Register(&awsPlugin{})
}

/*---------------------- core logic ----------------------*/

type creds struct {
	Access string
	Secret string
	Region string
}

func runSet(profile string) error {
	in := creds{}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("AWS Access Key ID").Value(&in.Access).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("AWS Secret Access Key").
				EchoMode(huh.EchoModePassword).
				Value(&in.Secret).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("Default Region (e.g. us-east-1)").Value(&in.Region).Validate(huh.ValidateNotEmpty()),
		),
	)

	if err := form.Run(); err != nil {
		return err // user cancelled
	}

	// Ensure ~/.aws exists
	if err := os.MkdirAll(filepath.Dir(credentialsPath()), 0o700); err != nil {
		return err
	}

	/* ---- credentials file ---- */
	credINI := ini.Empty()
	if fi, err := os.Stat(credentialsPath()); err == nil && fi.Size() > 0 {
		credINI, _ = ini.Load(credentialsPath())
	}
	csec := credINI.Section(profile)
	csec.Key("aws_access_key_id").SetValue(in.Access)
	csec.Key("aws_secret_access_key").SetValue(in.Secret)
	if err := credINI.SaveTo(credentialsPath()); err != nil {
		return err
	}

	/* ---- config file ---- */
	cfgINI := ini.Empty()
	if fi, err := os.Stat(configPath()); err == nil && fi.Size() > 0 {
		cfgINI, _ = ini.Load(configPath())
	}
	cfgSec := cfgINI.Section("profile " + profile)
	cfgSec.Key("region").SetValue(in.Region)
	if err := cfgINI.SaveTo(configPath()); err != nil {
		return err
	}

	fmt.Printf("✅ Credentials saved to %s (profile %q)\n", credentialsPath(), profile)
	return nil
}

func runExport(profile string) error {
	credINI, err := ini.Load(credentialsPath())
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}
	sec := credINI.Section(profile)
	id := sec.Key("aws_access_key_id").String()
	secret := sec.Key("aws_secret_access_key").String()
	if id == "" || secret == "" {
		return fmt.Errorf("profile %q not found in %s", profile, credentialsPath())
	}

	cfgINI, _ := ini.Load(configPath())
	region := cfgINI.Section("profile " + profile).Key("region").String()

	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", id)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", secret)
	if region != "" {
		fmt.Printf("export AWS_DEFAULT_REGION=%s\n", region)
	}
	return nil
}
