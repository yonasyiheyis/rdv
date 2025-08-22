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

type pgProfile struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type pgConfig struct {
	Profiles map[string]pgProfile `yaml:"profiles"`
}

func newPostgresCmd() *cobra.Command {
	var profile string
	var testConn bool

	pgCmd := &cobra.Command{
		Use:   "postgres",
		Short: "Manage PostgreSQL profiles",
	}

	// ------- set-config -------
	var noPrompt bool
	var inHost, inPort, inDB, inUser, inPass string

	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set PostgreSQL connection info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgSetConfig(profile, testConn, noPrompt, inHost, inPort, inDB, inUser, inPass)
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

	// ------- modify -------
	var modNoPrompt bool
	var mHost, mPort, mDB, mUser, mPass string

	modCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing Postgres profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgModify(profile, testConn, modNoPrompt, mHost, mPort, mDB, mUser, mPass)
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

	// ------- delete -----------
	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Postgres profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgDelete(profile)
		},
	}
	delCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	// ------- export -----------
	var envPath string

	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print DATABASE_URL (and PG* vars) for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgExport(profile, envPath)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	fflags.AddEnvFileFlag(expCmd.Flags(), &envPath) // --env-file/-o flag

	// ------- list ------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Postgres profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgList()
		},
	}

	// ------- show ------------
	var showName string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show Postgres profile (redacted)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgShow(showName)
		},
	}
	showCmd.Flags().StringVarP(&showName, "profile", "p", "default", "profile name")

	pgCmd.AddCommand(setCmd, modCmd, delCmd, expCmd, listCmd, showCmd)
	return pgCmd
}

/* ---------------- set-config ---------------- */
func pgSetConfig(profile string, testConn, noPrompt bool, host, port, db, user, pass string) error {
	var in pgProfile

	if noPrompt || !cli.IsTerminal() {
		if host == "" || port == "" || db == "" || user == "" || pass == "" {
			return fmt.Errorf("missing required flags: --host --port --dbname --user --password")
		}
		in = pgProfile{Host: host, Port: port, DBName: db, User: user, Password: pass}
	} else {
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Host").Value(&in.Host).Placeholder("localhost").Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Port").Value(&in.Port).Placeholder("5432").Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Database").Value(&in.DBName).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Username").Value(&in.User).Validate(huh.ValidateNotEmpty()),
				huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&in.Password).Validate(huh.ValidateNotEmpty()),
			),
		)
		if err := f.Run(); err != nil {
			return err
		}
	}

	// ensure dir
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return err
	}

	// read, merge, save
	cfg := pgConfig{Profiles: map[string]pgProfile{}}
	if b, err := os.ReadFile(postgresPath()); err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	cfg.Profiles[profile] = in
	out, _ := yaml.Marshal(cfg)
	if err := os.WriteFile(postgresPath(), out, 0o600); err != nil {
		return err
	}

	logger.L.Infow("✅ PostgreSQL profile saved", "profile", profile, "file", postgresPath())
	if testConn {
		return testPgConn(in)
	}
	return nil
}

/* ---------------- modify ---------------- */
func pgModify(profile string, testConn, noPrompt bool, host, port, db, user, pass string) error {
	cfg, err := loadPgConfig()
	if err != nil {
		return err
	}
	in := cfg.Profiles[profile] // zero if missing

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
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg.Profiles[profile] = in
	if err := savePgConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("pg profile modified", "profile", profile)
	fmt.Printf("✅ Updated Postgres profile %q\n", profile)

	if testConn {
		return testPgConn(in)
	}
	return nil
}

/* ---------------- delete ---------------- */

func pgDelete(profile string) error {
	ok, err := ui.Confirm(fmt.Sprintf("Delete Postgres profile %q?", profile))
	if err != nil || !ok {
		return err
	}

	cfg, err := loadPgConfig()
	if err != nil {
		return err
	}

	delete(cfg.Profiles, profile)
	if err := savePgConfig(cfg); err != nil {
		return err
	}

	logger.L.Infow("pg profile deleted", "profile", profile)
	fmt.Printf("🗑️  Deleted Postgres profile %q\n", profile)
	return nil
}

/* ---------------- export ---------------- */

func pgExport(profile string, envPath string) error {
	b, err := os.ReadFile(postgresPath())
	if err != nil {
		return fmt.Errorf("could not read %s: %w", postgresPath(), err)
	}
	var cfg pgConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return err
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("profile %q not found in %s", profile, postgresPath())
	}
	// Build URL (disable SSL by default for local dev)
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DBName)

	vars := map[string]string{
		"PG_DATABASE_URL": url,
		"PGHOST":          p.Host,
		"PGPORT":          p.Port,
		"PGUSER":          p.User,
		"PGPASSWORD":      p.Password,
		"PGDATABASE":      p.DBName,
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

func loadPgConfig() (pgConfig, error) {
	cfg := pgConfig{Profiles: map[string]pgProfile{}}
	b, err := os.ReadFile(postgresPath())
	if err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	return cfg, nil
}

func savePgConfig(cfg pgConfig) error {
	out, _ := yaml.Marshal(cfg)
	return os.WriteFile(postgresPath(), out, 0o600)
}

/* ---------------- list ---------------- */
func pgList() error {
	cfg, _ := loadPgConfig()
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

/* ---------------- show ---------------- */
func pgShow(name string) error {
	cfg, _ := loadPgConfig()
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	payload := map[string]any{
		"profile":  name,
		"host":     p.Host,
		"port":     p.Port,
		"dbname":   p.DBName,
		"user":     p.User,
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
	return nil
}
