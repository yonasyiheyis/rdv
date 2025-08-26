package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"

	"github.com/yonasyiheyis/rdv/internal/cli"
	"github.com/yonasyiheyis/rdv/internal/envfile"
	fflags "github.com/yonasyiheyis/rdv/internal/flags"
	"github.com/yonasyiheyis/rdv/internal/logger"
	"github.com/yonasyiheyis/rdv/internal/plugin"
	iprint "github.com/yonasyiheyis/rdv/internal/print"
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
	var setNoPrompt bool
	var inAccess, inSecret, inRegion string

	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set AWS credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSet(setProfile, setTestConn, setNoPrompt, inAccess, inSecret, inRegion)
		},
	}
	fflags.AddNoPromptFlag(setCmd.Flags(), &setNoPrompt)
	setCmd.Flags().StringVarP(&setProfile, "profile", "p", "default", "AWS profile")
	setCmd.Flags().BoolVar(&setTestConn, "test-conn", false, "validate credentials with STS after saving")
	setCmd.Flags().StringVar(&inAccess, "access-key", "", "AWS access key id")
	setCmd.Flags().StringVar(&inSecret, "secret-key", "", "AWS secret access key")
	setCmd.Flags().StringVar(&inRegion, "region", "", "AWS default region (e.g. us-east-1)")

	// -------- modify ----------------
	var modProfile string
	var modTestConn bool
	var modNoPrompt bool
	var modAccess, modSecret, modRegion string
	modifyCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing AWS profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runModifyAWS(modProfile, modTestConn, modNoPrompt, modAccess, modSecret, modRegion)
		},
	}
	fflags.AddNoPromptFlag(modifyCmd.Flags(), &modNoPrompt)
	modifyCmd.Flags().StringVarP(&modProfile, "profile", "p", "default", "AWS profile")
	modifyCmd.Flags().BoolVar(&modTestConn, "test-conn", false, "validate credentials with STS after saving")
	modifyCmd.Flags().StringVar(&modAccess, "access-key", "", "AWS access key id")
	modifyCmd.Flags().StringVar(&modSecret, "secret-key", "", "AWS secret access key")
	modifyCmd.Flags().StringVar(&modRegion, "region", "", "AWS default region")

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
	var envPath string
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Print AWS_* export lines for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(expProfile, envPath)
		},
	}
	exportCmd.Flags().StringVarP(&expProfile, "profile", "p", "default", "AWS profile")
	fflags.AddEnvFileFlag(exportCmd.Flags(), &envPath) // --env-file/-o flag

	// -------- list ----------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List AWS profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListAWS()
		},
	}

	// -------- show ----------------
	var showProfile string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show one AWS profile (redacted)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runShowAWS(showProfile)
		},
	}
	showCmd.Flags().StringVarP(&showProfile, "profile", "p", "default", "AWS profile")

	awsCmd.AddCommand(setCmd, modifyCmd, deleteCmd, exportCmd, listCmd, showCmd)
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

// ExportVars returns the map of AWS_* variables for the given profile.
func ExportVars(profile string) (map[string]string, error) {
	credINI, err := ini.Load(credentialsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	sec := credINI.Section(profile)
	id := strings.TrimSpace(sec.Key("aws_access_key_id").String())
	secret := strings.TrimSpace(sec.Key("aws_secret_access_key").String())
	if id == "" || secret == "" {
		return nil, fmt.Errorf("profile %q not found in %s", profile, credentialsPath())
	}

	cfgINI, _ := ini.Load(configPath())
	region := strings.TrimSpace(cfgINI.Section("profile " + profile).Key("region").String())

	vars := map[string]string{
		"AWS_ACCESS_KEY_ID":     id,
		"AWS_SECRET_ACCESS_KEY": secret,
	}
	if region != "" {
		vars["AWS_DEFAULT_REGION"] = region
	}
	return vars, nil
}

// ---------- command impls ----------

func runSet(profile string, testConn, noPrompt bool, access, secret, region string) error {
	in := credsInput{}

	if noPrompt || !cli.IsTerminal() {
		if access == "" || secret == "" || region == "" {
			return fmt.Errorf("missing required flags: --access-key, --secret-key, --region")
		}
		in.AccessKey, in.SecretKey, in.Region = access, secret, region
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("AWS Access Key ID").Value(&in.AccessKey).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("AWS Secret Access Key").EchoMode(huh.EchoModePassword).Value(&in.SecretKey).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Default Region (e.g. us-east-1)").Value(&in.Region).Validate(huh.ValidateNotEmpty()),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
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

func runModifyAWS(profile string, testConn, noPrompt bool, access, secret, region string) error {
	current, err := loadAWSProfile(profile)
	if err != nil {
		return err
	}
	in := current

	if noPrompt || !cli.IsTerminal() {
		if access != "" {
			in.AccessKey = access
		}
		if secret != "" {
			in.SecretKey = secret
		}
		if region != "" {
			in.Region = region
		}
		if in.AccessKey == "" || in.SecretKey == "" || in.Region == "" {
			return fmt.Errorf("missing values; provide all with flags or run interactively")
		}
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("AWS Access Key ID").Value(&in.AccessKey).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("AWS Secret Access Key").EchoMode(huh.EchoModePassword).Value(&in.SecretKey).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Default Region").Value(&in.Region).Validate(huh.ValidateNotEmpty()),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
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

func runExport(profile string, envPath string) error {
	vars, err := ExportVars(profile)
	if err != nil {
		return err
	}

	if envPath != "" { // write/merge to .env
		if err := envfile.WriteEnv(envPath, vars); err != nil {
			return err
		}
		if iprint.JSON {
			return iprint.Out(map[string]any{
				"written": len(vars),
				"path":    envPath,
				"vars":    vars,
			})
		}
		fmt.Printf("✅ wrote %d vars to %s\n", len(vars), envPath)
		return nil
	}

	if iprint.JSON {
		return iprint.Out(vars)
	}

	for k, v := range vars { // fallback: print exports
		fmt.Printf("export %s=%s\n", k, v)
	}
	return nil
}

func runListAWS() error {
	namesSet := map[string]struct{}{}

	if credINI, err := ini.Load(credentialsPath()); err == nil {
		for _, s := range credINI.Sections() {
			n := s.Name()
			if n == "DEFAULT" || n == "" {
				continue
			}
			namesSet[n] = struct{}{}
		}
	}
	if cfgINI, err := ini.Load(configPath()); err == nil {
		for _, s := range cfgINI.Sections() {
			n := strings.TrimPrefix(s.Name(), "profile ")
			if n == "DEFAULT" || n == "" {
				continue
			}
			namesSet[n] = struct{}{}
		}
	}

	names := make([]string, 0, len(namesSet))
	for n := range namesSet {
		names = append(names, n)
	}
	sort.Strings(names)

	if iprint.JSON {
		return iprint.Out(map[string]any{"profiles": names})
	}
	if len(names) == 0 {
		fmt.Println("(no profiles)")
		return nil
	}
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

func runShowAWS(profile string) error {
	cur, err := loadAWSProfile(profile)
	if err != nil {
		return err
	}
	if cur.AccessKey == "" && cur.SecretKey == "" && cur.Region == "" {
		return fmt.Errorf("profile %q not found", profile)
	}

	payload := map[string]any{
		"profile":    profile,
		"access_key": iprint.Redact(cur.AccessKey),
		"secret_key": iprint.Redact(cur.SecretKey),
		"region":     cur.Region,
	}
	if iprint.JSON {
		return iprint.Out(payload)
	}

	fmt.Printf("profile: %s\n", profile)
	fmt.Printf("  access_key: %s\n", iprint.Redact(cur.AccessKey))
	fmt.Printf("  secret_key: %s\n", iprint.Redact(cur.SecretKey))
	fmt.Printf("  region    : %s\n", cur.Region)
	return nil
}
