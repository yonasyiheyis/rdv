package github

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yonasyiheyis/rdv/internal/cli"
	"github.com/yonasyiheyis/rdv/internal/envfile"
	fflags "github.com/yonasyiheyis/rdv/internal/flags"
	"github.com/yonasyiheyis/rdv/internal/logger"
	"github.com/yonasyiheyis/rdv/internal/plugin"
	iprint "github.com/yonasyiheyis/rdv/internal/print"
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

	// -------- set-config ------------
	var noPrompt bool
	var token, apiBase string

	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set a GitHub token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghSetConfig(profile, testConn, noPrompt, token, apiBase)
		},
	}
	fflags.AddNoPromptFlag(setCmd.Flags(), &noPrompt)
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	setCmd.Flags().BoolVar(&testConn, "test-conn", false, "call GitHub API to validate token")
	setCmd.Flags().StringVar(&token, "token", "", "PAT or GitHub token")
	setCmd.Flags().StringVar(&apiBase, "api-base", "", "GitHub API base (optional, e.g. https://api.github.com/)")

	// -------- modify ----------------
	var modNoPrompt bool
	var mToken, mAPI string

	modCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing GitHub token profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghModify(profile, testConn, modNoPrompt, mToken, mAPI)
		},
	}
	fflags.AddNoPromptFlag(modCmd.Flags(), &modNoPrompt)
	modCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	modCmd.Flags().BoolVar(&testConn, "test-conn", false, "validate after saving")
	modCmd.Flags().StringVar(&mToken, "token", "", "PAT or GitHub token")
	modCmd.Flags().StringVar(&mAPI, "api-base", "", "GitHub API base (optional)")

	// -------- delete ----------------
	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GitHub token profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghDelete(profile)
		},
	}
	delCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	// -------- export ----------------
	var envPath string

	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print GITHUB_TOKEN export line (and optional vars)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ghExport(profile, envPath)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	fflags.AddEnvFileFlag(expCmd.Flags(), &envPath) // --env-file/-o flag

	// -------- list ----------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List GitHub profiles",
		RunE:  func(cmd *cobra.Command, _ []string) error { return ghList() },
	}

	// -------- show ----------------
	var showName string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show GitHub profile (redacted)",
		RunE:  func(cmd *cobra.Command, _ []string) error { return ghShow(showName) },
	}
	showCmd.Flags().StringVarP(&showName, "profile", "p", "default", "profile name")

	ghCmd.AddCommand(setCmd, modCmd, delCmd, expCmd, listCmd, showCmd)
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

func ghSetConfig(profile string, testConn, noPrompt bool, tok, api string) error {
	var p ghProfile

	if noPrompt || !cli.IsTerminal() {
		if tok == "" {
			return fmt.Errorf("missing required flag: --token")
		}
		p.Token = tok
		p.APIBase = api
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("GitHub Token").EchoMode(huh.EchoModePassword).Value(&p.Token).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("API Base URL (optional)").Value(&p.APIBase).Placeholder("https://api.github.com/"),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]ghProfile{}
	}
	cfg.Profiles[profile] = p
	if err := saveCfg(cfg); err != nil {
		return err
	}

	if testConn {
		if err := testToken(&p); err != nil {
			return err
		}
		cfg.Profiles[profile] = p // user may be populated by testToken
		_ = saveCfg(cfg)
	}

	logger.L.Infow("github token saved", "profile", profile)
	fmt.Printf("✅ GitHub profile %q saved to %s\n", profile, cfgPath())
	return nil
}

func ghModify(profile string, testConn, noPrompt bool, tok, api string) error {
	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	p := cfg.Profiles[profile] // zero value if missing

	if noPrompt || !cli.IsTerminal() {
		if tok != "" {
			p.Token = tok
		}
		if api != "" {
			p.APIBase = api
		}
		if p.Token == "" {
			return fmt.Errorf("missing values; provide --token or run interactively")
		}
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("GitHub Token").EchoMode(huh.EchoModePassword).Value(&p.Token).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("API Base URL (optional)").Value(&p.APIBase),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg.Profiles[profile] = p
	if err := saveCfg(cfg); err != nil {
		return err
	}

	if testConn {
		if err := testToken(&p); err != nil {
			return err
		}
		cfg.Profiles[profile] = p
		_ = saveCfg(cfg)
	}

	logger.L.Infow("github profile modified", "profile", profile)
	fmt.Printf("✅ Updated GitHub profile %q\n", profile)
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

func ghExport(profile string, envPath string) error {
	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("profile %q not found in %s", profile, cfgPath())
	}

	vars := map[string]string{
		"GITHUB_TOKEN": p.Token,
	}
	if p.APIBase != "" {
		vars["GITHUB_API_BASE"] = p.APIBase
	}
	if p.User != "" {
		vars["GITHUB_USER"] = p.User
	}

	if envPath != "" { // write/merge to .env
		if err := envfile.WriteEnv(envPath, vars); err != nil {
			return err
		}
		fmt.Printf("✅ wrote %d vars to %s\n", len(vars), envPath)
		return nil
	}

	for k, v := range vars { // fallback: print exports
		fmt.Printf("export %s=%s\n", k, v)
	}
	return nil
}

func ghList() error {
	cfg, _ := loadCfg()
	if len(cfg.Profiles) == 0 {
		fmt.Println("(no profiles)")
		return nil
	}
	for name := range cfg.Profiles {
		fmt.Println(name)
	}
	return nil
}

func ghShow(name string) error {
	cfg, _ := loadCfg()
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	fmt.Printf("profile: %s\n", name)
	fmt.Printf("  token   : %s\n", iprint.Redact(p.Token))
	if p.APIBase != "" {
		fmt.Printf("  api_base: %s\n", p.APIBase)
	}
	if p.User != "" {
		fmt.Printf("  user    : %s\n", p.User)
	}
	return nil
}
