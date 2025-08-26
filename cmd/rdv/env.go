package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yonasyiheyis/rdv/internal/envfile"
	awsp "github.com/yonasyiheyis/rdv/internal/plugins/aws"
	dbp "github.com/yonasyiheyis/rdv/internal/plugins/db"
	ghp "github.com/yonasyiheyis/rdv/internal/plugins/github"
	iprint "github.com/yonasyiheyis/rdv/internal/print"
)

func newEnvCmd() *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Environment helpers",
	}

	envCmd.AddCommand(newEnvExportCmd())
	return envCmd
}

func newEnvExportCmd() *cobra.Command {
	var sets []string // repeated --set like: aws:dev, db.postgres:dev, db.mysql:ci, github:bot
	var envPath string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Merge and export variables from multiple profiles",
		Example: `  rdv env export --set aws:dev --set db.postgres:dev
  rdv env export --set aws:dev --set github:bot --json
  rdv env export --set db.mysql:ci --env-file .env.ci`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(sets) == 0 {
				return fmt.Errorf("at least one --set is required (e.g., --set aws:dev)")
			}

			merged := map[string]string{}

			for _, s := range sets {
				parts := strings.SplitN(s, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --set %q, expected <plugin>[:subplugin]:<profile>", s)
				}
				target, prof := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

				vars, err := exportFor(target, prof)
				if err != nil {
					return err
				}
				for k, v := range vars {
					// last writer wins on conflicts
					merged[k] = v
				}
			}

			// Output
			if iprint.JSON && envPath == "" {
				return iprint.Out(merged)
			}
			if envPath != "" {
				if err := envfile.WriteEnv(envPath, merged); err != nil {
					return err
				}
				if iprint.JSON {
					return iprint.Out(map[string]any{
						"written": len(merged),
						"path":    envPath,
						"vars":    merged,
					})
				}
				fmt.Printf("✅ wrote %d vars to %s\n", len(merged), envPath)
				return nil
			}

			// deterministic order for humans
			keys := make([]string, 0, len(merged))
			for k := range merged {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("export %s=%s\n", k, merged[k])
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&sets, "set", nil, "profile spec: aws:<name> | db.postgres:<name> | db.mysql:<name> | github:<name> (repeatable)")
	cmd.Flags().StringVarP(&envPath, "env-file", "o", "", "write/merge result to this .env file instead of printing")
	return cmd
}

// exportFor resolves a <plugin>[:subplugin] + profile into a map of env vars.
func exportFor(target, profile string) (map[string]string, error) {
	switch target {
	case "aws":
		return awsp.ExportVars(profile)
	case "db.postgres", "postgres", "pg":
		return dbp.PGExportVars(profile)
	case "db.mysql", "mysql":
		return dbp.MySQLExportVars(profile)
	case "github", "gh":
		return ghp.ExportVars(profile)
	default:
		return nil, fmt.Errorf("unknown target %q (expected aws|db.postgres|db.mysql|github)", target)
	}
}
