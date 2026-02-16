package db

import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yonasyiheyis/rdv/internal/cli"
	"github.com/yonasyiheyis/rdv/internal/envfile"
	"github.com/yonasyiheyis/rdv/internal/exitcodes"
	fflags "github.com/yonasyiheyis/rdv/internal/flags"
	"github.com/yonasyiheyis/rdv/internal/logger"
	iprint "github.com/yonasyiheyis/rdv/internal/print"
	"github.com/yonasyiheyis/rdv/internal/ui"
)

type redisProfile struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       string `yaml:"db"`
	TLS      bool   `yaml:"tls"`
}

type redisConfig struct {
	Profiles map[string]redisProfile `yaml:"profiles"`
}

func newRedisCmd() *cobra.Command {
	var profile string
	var testConn bool

	redisCmd := &cobra.Command{
		Use:   "redis",
		Short: "Manage Redis profiles",
	}

	// -------- set-config --------
	var noPrompt bool
	var inHost, inPort, inDB, inPass string
	var inTLS bool

	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set Redis connection info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return redisSetConfig(profile, testConn, noPrompt, inHost, inPort, inDB, inPass, inTLS)
		},
	}
	fflags.AddNoPromptFlag(setCmd.Flags(), &noPrompt)
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	setCmd.Flags().BoolVar(&testConn, "test-conn", false, "try to connect after saving")
	setCmd.Flags().StringVar(&inHost, "host", "", "Redis host")
	setCmd.Flags().StringVar(&inPort, "port", "", "Redis port")
	setCmd.Flags().StringVar(&inDB, "db", "", "Redis database number (0-15)")
	setCmd.Flags().StringVar(&inPass, "password", "", "Redis password")
	setCmd.Flags().BoolVar(&inTLS, "tls", false, "use TLS connection")

	// -------- modify ------------
	var modNoPrompt bool
	var mHost, mPort, mDB, mPass string
	var mTLS bool

	modCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing Redis profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return redisModify(profile, testConn, modNoPrompt, mHost, mPort, mDB, mPass, cmd.Flags().Changed("tls"), mTLS)
		},
	}
	fflags.AddNoPromptFlag(modCmd.Flags(), &modNoPrompt)
	modCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	modCmd.Flags().BoolVar(&testConn, "test-conn", false, "try to connect after saving")
	modCmd.Flags().StringVar(&mHost, "host", "", "Redis host")
	modCmd.Flags().StringVar(&mPort, "port", "", "Redis port")
	modCmd.Flags().StringVar(&mDB, "db", "", "Redis database number")
	modCmd.Flags().StringVar(&mPass, "password", "", "Redis password")
	modCmd.Flags().BoolVar(&mTLS, "tls", false, "use TLS connection")

	// -------- delete ------------
	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Redis profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return redisDelete(profile)
		},
	}
	delCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	// -------- export ------------
	var envPath string

	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print REDIS_* exports for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return redisExport(profile, envPath)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	fflags.AddEnvFileFlag(expCmd.Flags(), &envPath) // --env-file/-o flag

	// -------- list ------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Redis profiles",
		RunE:  func(cmd *cobra.Command, _ []string) error { return redisList() },
	}

	// -------- show ------------
	var showName string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show Redis profile (redacted)",
		RunE:  func(cmd *cobra.Command, _ []string) error { return redisShow(showName) },
	}
	showCmd.Flags().StringVarP(&showName, "profile", "p", "default", "profile name")

	redisCmd.AddCommand(setCmd, modCmd, delCmd, expCmd, listCmd, showCmd)
	return redisCmd
}

// ------------ helpers ------------

func loadRedisConfig() (redisConfig, error) {
	cfg := redisConfig{Profiles: map[string]redisProfile{}}
	if b, err := os.ReadFile(redisPath()); err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	return cfg, nil
}

func saveRedisConfig(cfg redisConfig) error {
	out, _ := yaml.Marshal(cfg)
	return os.WriteFile(redisPath(), out, 0o600)
}

// RedisExportVars builds the Redis env var map for a profile.
func RedisExportVars(profile string) (map[string]string, error) {
	cfg, err := loadRedisConfig()
	if err != nil {
		return nil, err
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return nil, exitcodes.New(exitcodes.ProfileNotFound,
			fmt.Sprintf("profile %q not found in %s", profile, redisPath()))
	}

	scheme := "redis"
	if p.TLS {
		scheme = "rediss"
	}

	url := fmt.Sprintf("%s://:%s@%s:%s/%s", scheme, p.Password, p.Host, p.Port, p.DB)
	if p.Password == "" {
		url = fmt.Sprintf("%s://%s:%s/%s", scheme, p.Host, p.Port, p.DB)
	}

	vars := map[string]string{
		"REDIS_URL":  url,
		"REDIS_HOST": p.Host,
		"REDIS_PORT": p.Port,
		"REDIS_DB":   p.DB,
		"REDIS_TLS":  fmt.Sprintf("%t", p.TLS),
	}
	if p.Password != "" {
		vars["REDIS_PASSWORD"] = p.Password
	}
	return vars, nil
}

// ------------ command logic ------------

func redisSetConfig(profile string, testConn, noPrompt bool, host, port, db, pass string, tls bool) error {
	var in redisProfile

	if noPrompt || !cli.IsInteractive() {
		if host == "" || port == "" {
			return exitcodes.New(exitcodes.InvalidArgs, "missing required flags: --host and --port")
		}
		if db == "" {
			db = "0"
		}
		in = redisProfile{Host: host, Port: port, DB: db, Password: pass, TLS: tls}
	} else {
		in.DB = "0" // set default for interactive
		form := ui.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Host").Value(&in.Host).Validate(huh.ValidateNotEmpty()).Placeholder("localhost"),
				huh.NewInput().Title("Port").Value(&in.Port).Validate(huh.ValidateNotEmpty()).Placeholder("6379"),
				huh.NewInput().Title("Password (optional)").EchoMode(huh.EchoModePassword).Value(&in.Password),
				huh.NewInput().Title("Database").Value(&in.DB).Validate(huh.ValidateNotEmpty()).Placeholder("0"),
				huh.NewConfirm().Title("Use TLS?").Value(&in.TLS),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg, err := loadRedisConfig()
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]redisProfile{}
	}
	cfg.Profiles[profile] = in
	if err := saveRedisConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("redis profile saved", "profile", profile)
	fmt.Printf("✅ Redis profile %q saved to %s\n", profile, redisPath())

	if testConn {
		if err := testRedisConn(in); err != nil {
			return exitcodes.Wrap(exitcodes.ConnectionFailed, err)
		}
	}
	return nil
}

func redisModify(profile string, testConn, noPrompt bool, host, port, db, pass string, tlsChanged bool, tls bool) error {
	cfg, err := loadRedisConfig()
	if err != nil {
		return err
	}
	in := cfg.Profiles[profile] // zero if not found

	if noPrompt || !cli.IsInteractive() {
		if host != "" {
			in.Host = host
		}
		if port != "" {
			in.Port = port
		}
		if db != "" {
			in.DB = db
		}
		if pass != "" {
			in.Password = pass
		}
		if tlsChanged {
			in.TLS = tls
		}
		if in.Host == "" || in.Port == "" || in.DB == "" {
			return exitcodes.New(exitcodes.InvalidArgs, "missing values; provide all with flags or run interactively")
		}
	} else {
		if in.DB == "" {
			in.DB = "0"
		}
		form := ui.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Host").Value(&in.Host).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Port").Value(&in.Port).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Password (optional)").EchoMode(huh.EchoModePassword).Value(&in.Password),
				huh.NewInput().Title("Database").Value(&in.DB).Validate(huh.ValidateNotEmpty()),
				huh.NewConfirm().Title("Use TLS?").Value(&in.TLS),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg.Profiles[profile] = in
	if err := saveRedisConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("redis profile modified", "profile", profile)
	fmt.Printf("✅ Updated Redis profile %q\n", profile)

	if testConn {
		if err := testRedisConn(in); err != nil {
			return exitcodes.Wrap(exitcodes.ConnectionFailed, err)
		}
	}
	return nil
}

func redisDelete(profile string) error {
	ok, err := ui.Confirm(fmt.Sprintf("Delete Redis profile %q?", profile))
	if err != nil || !ok {
		return err
	}

	cfg, err := loadRedisConfig()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, profile)
	if err := saveRedisConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("redis profile deleted", "profile", profile)
	fmt.Printf("🗑️  Deleted Redis profile %q\n", profile)
	return nil
}

func redisExport(profile string, envPath string) error {
	vars, err := RedisExportVars(profile)
	if err != nil {
		return err
	}

	if envPath != "" { // write/merge to .env
		if err := envfile.WriteEnv(envPath, vars); err != nil {
			return exitcodes.Wrap(exitcodes.EnvWriteFailed, err)
		}
		if iprint.JSON {
			if err := iprint.Out(map[string]any{
				"written": len(vars),
				"path":    envPath,
				"vars":    vars,
			}); err != nil {
				return exitcodes.Wrap(exitcodes.JSONError, err)
			}
			return nil
		}
		fmt.Printf("✅ wrote %d vars to %s\n", len(vars), envPath)
		return nil
	}

	if iprint.JSON {
		if err := iprint.Out(vars); err != nil {
			return exitcodes.Wrap(exitcodes.JSONError, err)
		}
		return nil
	}

	for k, v := range vars { // fallback: print exports
		fmt.Printf("export %s=%s\n", k, v)
	}
	return nil
}

func redisList() error {
	cfg, _ := loadRedisConfig()
	if len(cfg.Profiles) == 0 {
		if iprint.JSON {
			return iprint.Out(map[string]any{"profiles": []string{}})
		}
		fmt.Println("(no profiles)")
		return nil
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	if iprint.JSON {
		return iprint.Out(map[string]any{"profiles": names})
	}
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

func redisShow(name string) error {
	cfg, _ := loadRedisConfig()
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	payload := map[string]any{
		"profile":  name,
		"host":     p.Host,
		"port":     p.Port,
		"db":       p.DB,
		"tls":      p.TLS,
		"password": iprint.Redact(p.Password),
	}

	if iprint.JSON {
		return iprint.Out(payload)
	}
	fmt.Printf("profile: %s\n", name)
	fmt.Printf("  host    : %s\n", p.Host)
	fmt.Printf("  port    : %s\n", p.Port)
	fmt.Printf("  db      : %s\n", p.DB)
	fmt.Printf("  tls     : %t\n", p.TLS)
	fmt.Printf("  password: %s\n", iprint.Redact(p.Password))
	return nil
}
