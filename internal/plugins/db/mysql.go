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
	fflags "github.com/yonasyiheyis/rdv/internal/flags"
	"github.com/yonasyiheyis/rdv/internal/logger"
	iprint "github.com/yonasyiheyis/rdv/internal/print"
	"github.com/yonasyiheyis/rdv/internal/ui"
)

type mysqlProfile struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	Params   string `yaml:"params,omitempty"` // optional DSN params
}

type mysqlConfig struct {
	Profiles map[string]mysqlProfile `yaml:"profiles"`
}

func newMySQLCmd() *cobra.Command {
	var profile string
	var testConn bool

	mysqlCmd := &cobra.Command{
		Use:   "mysql",
		Short: "Manage MySQL/MariaDB profiles",
	}

	// -------- set-config --------
	var noPrompt bool
	var inHost, inPort, inDB, inUser, inPass, inParams string

	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set MySQL connection info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlSetConfig(profile, testConn, noPrompt, inHost, inPort, inDB, inUser, inPass, inParams)
		},
	}
	fflags.AddNoPromptFlag(setCmd.Flags(), &noPrompt)
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	setCmd.Flags().BoolVar(&testConn, "test-conn", false, "try to connect after saving")
	setCmd.Flags().StringVar(&inHost, "host", "", "db host")
	setCmd.Flags().StringVar(&inPort, "port", "", "db port")
	setCmd.Flags().StringVar(&inDB, "dbname", "", "db name")
	setCmd.Flags().StringVar(&inUser, "user", "", "db user")
	setCmd.Flags().StringVar(&inPass, "password", "", "db password")
	setCmd.Flags().StringVar(&inParams, "params", "", "extra DSN params (e.g. parseTime=true)")

	// -------- modify ------------
	var modNoPrompt bool
	var mHost, mPort, mDB, mUser, mPass, mParams string

	modCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing MySQL profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlModify(profile, testConn, modNoPrompt, mHost, mPort, mDB, mUser, mPass, mParams)
		},
	}
	fflags.AddNoPromptFlag(modCmd.Flags(), &modNoPrompt)
	modCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	modCmd.Flags().BoolVar(&testConn, "test-conn", false, "try to connect after saving")
	modCmd.Flags().StringVar(&mHost, "host", "", "db host")
	modCmd.Flags().StringVar(&mPort, "port", "", "db port")
	modCmd.Flags().StringVar(&mDB, "dbname", "", "db name")
	modCmd.Flags().StringVar(&mUser, "user", "", "db user")
	modCmd.Flags().StringVar(&mPass, "password", "", "db password")
	modCmd.Flags().StringVar(&mParams, "params", "", "extra DSN params")

	// -------- delete ------------
	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a MySQL profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlDelete(profile)
		},
	}
	delCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	// -------- export ------------
	var envPath string

	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print DATABASE_URL and MYSQL_* exports for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlExport(profile, envPath)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	fflags.AddEnvFileFlag(expCmd.Flags(), &envPath) // --env-file/-o flag

	// -------- list ------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List MySQL profiles",
		RunE:  func(cmd *cobra.Command, _ []string) error { return mysqlList() },
	}

	// -------- show ------------
	var showName string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show MySQL profile (redacted)",
		RunE:  func(cmd *cobra.Command, _ []string) error { return mysqlShow(showName) },
	}
	showCmd.Flags().StringVarP(&showName, "profile", "p", "default", "profile name")

	mysqlCmd.AddCommand(setCmd, modCmd, delCmd, expCmd, listCmd, showCmd)
	return mysqlCmd
}

// ------------ helpers ------------

func loadMySQLConfig() (mysqlConfig, error) {
	cfg := mysqlConfig{Profiles: map[string]mysqlProfile{}}
	if b, err := os.ReadFile(mysqlPath()); err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	return cfg, nil
}

func saveMySQLConfig(cfg mysqlConfig) error {
	out, _ := yaml.Marshal(cfg)
	return os.WriteFile(mysqlPath(), out, 0o600)
}

// ------------ command logic ------------

func mysqlSetConfig(profile string, testConn, noPrompt bool, host, port, db, user, pass, params string) error {
	var in mysqlProfile

	if noPrompt || !cli.IsTerminal() {
		if host == "" || port == "" || db == "" || user == "" || pass == "" {
			return fmt.Errorf("missing required flags: --host --port --dbname --user --password")
		}
		in = mysqlProfile{Host: host, Port: port, DBName: db, User: user, Password: pass, Params: params}
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Host").Value(&in.Host).Validate(huh.ValidateNotEmpty()).Placeholder("localhost"),
				huh.NewInput().Title("Port").Value(&in.Port).Validate(huh.ValidateNotEmpty()).Placeholder("3306"),
				huh.NewInput().Title("Database").Value(&in.DBName).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Username").Value(&in.User).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&in.Password).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Params (optional)").Value(&in.Params).Placeholder("parseTime=true"),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg, err := loadMySQLConfig()
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]mysqlProfile{}
	}
	cfg.Profiles[profile] = in
	if err := saveMySQLConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("mysql profile saved", "profile", profile)
	fmt.Printf("✅ MySQL profile %q saved to %s\n", profile, mysqlPath())

	if testConn {
		return testMySQLConn(in)
	}
	return nil
}

func mysqlModify(profile string, testConn, noPrompt bool, host, port, db, user, pass, params string) error {
	cfg, err := loadMySQLConfig()
	if err != nil {
		return err
	}
	in := cfg.Profiles[profile] // zero if not found

	if noPrompt || !cli.IsTerminal() {
		if host != "" {
			in.Host = host
		}
		if port != "" {
			in.Port = port
		}
		if db != "" {
			in.DBName = db
		}
		if user != "" {
			in.User = user
		}
		if pass != "" {
			in.Password = pass
		}
		if params != "" {
			in.Params = params
		}
		if in.Host == "" || in.Port == "" || in.DBName == "" || in.User == "" || in.Password == "" {
			return fmt.Errorf("missing values; provide all with flags or run interactively")
		}
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Host").Value(&in.Host).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Port").Value(&in.Port).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Database").Value(&in.DBName).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Username").Value(&in.User).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&in.Password).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Params (optional)").Value(&in.Params),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg.Profiles[profile] = in
	if err := saveMySQLConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("mysql profile modified", "profile", profile)
	fmt.Printf("✅ Updated MySQL profile %q\n", profile)

	if testConn {
		return testMySQLConn(in)
	}
	return nil
}

func mysqlDelete(profile string) error {
	ok, err := ui.Confirm(fmt.Sprintf("Delete MySQL profile %q?", profile))
	if err != nil || !ok {
		return err
	}

	cfg, err := loadMySQLConfig()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, profile)
	if err := saveMySQLConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("mysql profile deleted", "profile", profile)
	fmt.Printf("🗑️  Deleted MySQL profile %q\n", profile)
	return nil
}

func mysqlExport(profile string, envPath string) error {
	cfg, err := loadMySQLConfig()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("profile %q not found in %s", profile, mysqlPath())
	}

	dsn := buildMySQLDSN(p)
	url := buildMySQLURL(p)

	vars := map[string]string{
		"MYSQL_DATABASE_URL": url,
		"MYSQL_DSN":          dsn,
		"MYSQL_HOST":         p.Host,
		"MYSQL_PORT":         p.Port,
		"MYSQL_USER":         p.User,
		"MYSQL_PASSWORD":     p.Password,
		"MYSQL_DATABASE":     p.DBName,
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

func mysqlList() error {
	cfg, _ := loadMySQLConfig()
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

func mysqlShow(name string) error {
	cfg, _ := loadMySQLConfig()
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	payload := map[string]any{
		"profile": name,
		"host":    p.Host,
		"port":    p.Port,
		"dbname":  p.DBName,
		"user":    p.User,
		"params":  p.Params,
		// Redact
		"password": iprint.Redact(p.Password),
	}

	if iprint.JSON {
		return iprint.Out(payload)
	}
	fmt.Printf("profile: %s\n", name)
	fmt.Printf("  host    : %s\n", p.Host)
	fmt.Printf("  port    : %s\n", p.Port)
	fmt.Printf("  dbname  : %s\n", p.DBName)
	fmt.Printf("  user    : %s\n", p.User)
	fmt.Printf("  password: %s\n", iprint.Redact(p.Password))
	if p.Params != "" {
		fmt.Printf("  params  : %s\n", p.Params)
	}
	return nil
}
