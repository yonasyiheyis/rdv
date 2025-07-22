package db

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yonasyiheyis/rdv/internal/logger"
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

	pgCmd := &cobra.Command{
		Use:   "postgres",
		Short: "Manage PostgreSQL profiles",
	}

	// ------- set-config -------
	setCmd := &cobra.Command{
		Use:   "set-config",
		Short: "Interactively set PostgreSQL connection info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgSetConfig(profile)
		},
	}
	setCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	// ------- export -----------
	expCmd := &cobra.Command{
		Use:   "export",
		Short: "Print DATABASE_URL (and PG* vars) for a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pgExport(profile)
		},
	}
	expCmd.Flags().StringVarP(&profile, "profile", "p", "default", "profile name")

	pgCmd.AddCommand(setCmd, expCmd)
	return pgCmd
}

/* ---------------- set-config ---------------- */

func pgSetConfig(profile string) error {
	// interactive prompts
	var in pgProfile
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
		return err // cancelled
	}

	// ensure dir
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return err
	}

	// read existing file if exists
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
	return nil
}

/* ---------------- export ---------------- */

func pgExport(profile string) error {
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

	fmt.Printf("export DATABASE_URL=\"%s\"\n", url)
	fmt.Printf("export PGHOST=%s\n", p.Host)
	fmt.Printf("export PGPORT=%s\n", p.Port)
	fmt.Printf("export PGUSER=%s\n", p.User)
	fmt.Printf("export PGPASSWORD=%s\n", p.Password)
	fmt.Printf("export PGDATABASE=%s\n", p.DBName)
	return nil
}
