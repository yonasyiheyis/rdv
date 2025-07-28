package db

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yonasyiheyis/rdv/internal/logger"
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
	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set MySQL connection info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlSetConfig(profile, testConn)
		},
	}
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	setCmd.Flags().BoolVar(&testConn, "test-conn", false, "try to connect after saving")

	// -------- modify ------------
	modCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify an existing MySQL profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlModify(profile, testConn)
		},
	}
	modCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")
	modCmd.Flags().BoolVar(&testConn, "test-conn", false, "try to connect after saving")

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
	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print DATABASE_URL and MYSQL_* exports for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mysqlExport(profile)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	mysqlCmd.AddCommand(setCmd, modCmd, delCmd, expCmd)
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

func mysqlSetConfig(profile string, testConn bool) error {
	var in mysqlProfile

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

	cfg, err := loadMySQLConfig()
	if err != nil {
		return err
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

func mysqlModify(profile string, testConn bool) error {
	cfg, err := loadMySQLConfig()
	if err != nil {
		return err
	}
	in := cfg.Profiles[profile] // zero if not found

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

func mysqlExport(profile string) error {
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

	fmt.Printf("export DATABASE_URL=\"%s\"\n", url)
	fmt.Printf("export MYSQL_DSN=\"%s\"\n", dsn)
	fmt.Printf("export MYSQL_HOST=%s\n", p.Host)
	fmt.Printf("export MYSQL_PORT=%s\n", p.Port)
	fmt.Printf("export MYSQL_USER=%s\n", p.User)
	fmt.Printf("export MYSQL_PASSWORD=%s\n", p.Password)
	fmt.Printf("export MYSQL_DATABASE=%s\n", p.DBName)
	return nil
}
