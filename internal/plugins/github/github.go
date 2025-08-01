package github

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yonasyiheyis/rdv/internal/logger"
	"github.com/yonasyiheyis/rdv/internal/plugin"
	"github.com/yonasyiheyis/rdv/internal/ui"
)

// ---------- plugin wiring ----------

type ghPlugin struct{}

func (g *ghPlugin) Name() string { return "github" }

func (g *ghPlugin) Register(root *cobra.Command) {
	ghCmd := &cobra.Command{
		Use:   "github",
		Short: "Manage GitHub tokens",
	}

	var profile string
	var testConn bool

	// set-config
	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set a GitHub token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghSetConfig(profile, testConn)
		},
	}
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	setCmd.Flags().BoolVar(&testConn, "test-conn", false, "call GitHub API to validate token")

	// modify
	modCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing GitHub token profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghModify(profile, testConn)
		},
	}
	modCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	modCmd.Flags().BoolVar(&testConn, "test-conn", false, "validate after saving")

	// delete
	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GitHub token profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghDelete(profile)
		},
	}
	delCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	// export
	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print GITHUB_TOKEN export line (and optional vars)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghExport(profile)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	ghCmd.AddCommand(setCmd, modCmd, delCmd, expCmd)
	root.AddCommand(ghCmd)
}

func init() { plugin.Register(&ghPlugin{}) }

// ---------- data types ----------

type ghProfile struct {
	Token   string `yaml:"token"`
	APIBase string `yaml:"api_base,omitempty"` // e.g. GitHub Enterprise API URL
	User    string `yaml:"user,omitempty"`     // filled after test-conn
}

type ghConfig struct {
	Profiles map[string]ghProfile `yaml:"profiles"`
}

// ---------- helpers (load/save) ----------

func loadCfg() (ghConfig, error) {
	cfg := ghConfig{Profiles: map[string]ghProfile{}}
	if b, err := os.ReadFile(cfgPath()); err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	return cfg, nil
}

func saveCfg(cfg ghConfig) error {
	out, _ := yaml.Marshal(cfg)
	return os.WriteFile(cfgPath(), out, 0o600)
}

/* ------------ command impls ------------ */

func ghSetConfig(profile string, testConn bool) error {
	var p ghProfile

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("GitHub Token").EchoMode(huh.EchoModePassword).Value(&p.Token).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("API Base URL (optional)").Value(&p.APIBase).Placeholder("https://api.github.com/"),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	cfg.Profiles[profile] = p
	if err := saveCfg(cfg); err != nil {
		return err
	}

	logger.L.Infow("github token saved", "profile", profile)
	fmt.Printf("✅ GitHub profile %q saved to %s\n", profile, cfgPath())

	if testConn {
		if err := testToken(&p); err != nil {
			return err
		}
		// token valid, save user, etc.
		cfg.Profiles[profile] = p
		_ = saveCfg(cfg)
	}
	return nil
}

func ghModify(profile string, testConn bool) error {
	cfg, err := loadCfg()
	if err != nil {
		return err
	}

	p := cfg.Profiles[profile] // zero value if missing

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("GitHub Token").EchoMode(huh.EchoModePassword).Value(&p.Token).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("API Base URL (optional)").Value(&p.APIBase),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	cfg.Profiles[profile] = p
	if err := saveCfg(cfg); err != nil {
		return err
	}

	logger.L.Infow("github profile modified", "profile", profile)
	fmt.Printf("✅ Updated GitHub profile %q\n", profile)

	if testConn {
		if err := testToken(&p); err != nil {
			return err
		}
		cfg.Profiles[profile] = p
		_ = saveCfg(cfg)
	}

	return nil
}

func ghDelete(profile string) error {
	ok, err := ui.Confirm(fmt.Sprintf("Delete GitHub profile %q?", profile))
	if err != nil || !ok {
		return err
	}

	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, profile)
	if err := saveCfg(cfg); err != nil {
		return err
	}

	logger.L.Infow("github profile deleted", "profile", profile)
	fmt.Printf("🗑️  Deleted GitHub profile %q\n", profile)
	return nil
}

func ghExport(profile string) error {
	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("profile %q not found in %s", profile, cfgPath())
	}

	fmt.Printf("export GITHUB_TOKEN=%s\n", p.Token)
	if p.APIBase != "" {
		fmt.Printf("export GITHUB_API_BASE=%s\n", p.APIBase)
	}
	if p.User != "" {
		fmt.Printf("export GITHUB_USER=%s\n", p.User)
	}
	return nil
}
