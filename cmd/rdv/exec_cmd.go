package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	execenv "github.com/yonasyiheyis/rdv/internal/exec"
	"github.com/yonasyiheyis/rdv/internal/exitcodes"
)

func newExecCmd() *cobra.Command {
	var awsProf, pgProf, mysqlProf, ghProf string
	var noInherit bool

	cmd := &cobra.Command{
		Use:   "exec [-- command [args...]]",
		Short: "Run a command with env from selected profiles (ephemeral)",
		Long: `Run any command with env vars injected from saved profiles.

Examples:
  rdv exec --aws dev -- env | grep AWS_
  rdv exec --pg dev -- psql -c '\conninfo'
  rdv exec --aws dev --pg dev -- make test
  rdv exec --no-inherit --mysql ci -- /bin/sh -lc 'echo $MYSQL_DATABASE_URL'`,
		Args: cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			// Support separating flags from the command using "--"
			if split := c.ArgsLenAtDash(); split >= 0 {
				args = args[split:]
			}
			if len(args) == 0 {
				return exitcodes.New(exitcodes.InvalidArgs, "provide a command to run after --, e.g., rdv exec --aws dev -- env")
			}

			// Require at least one source; otherwise it's a no-op.
			if awsProf == "" && pgProf == "" && mysqlProf == "" && ghProf == "" {
				return fmt.Errorf("nothing to inject: pass one of --aws PROFILE, --pg PROFILE, --mysql PROFILE, or --github PROFILE")
			}

			envMap, err := execenv.BuildEnv(execenv.Options{
				AWS:       awsProf,
				Postgres:  pgProf,
				MySQL:     mysqlProf,
				GitHub:    ghProf,
				NoInherit: noInherit,
			})
			if err != nil {
				return err
			}

			// Convert map to []string form
			childEnv := make([]string, 0, len(envMap))
			for k, v := range envMap {
				childEnv = append(childEnv, k+"="+v)
			}

			// Spawn child process with inherited stdio
			child := exec.CommandContext(c.Context(), args[0], args[1:]...)
			child.Env = childEnv
			child.Stdin = os.Stdin
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr

			// Run and propagate error (exit code behavior will be refined in the exit-codes task)
			if err := child.Run(); err != nil {
				// If the child ran and failed, propagate its exit code exactly.
				if ee, ok := err.(*exec.ExitError); ok {
					return exitcodes.WithCode(ee.ExitCode())
				}
				// Otherwise: spawn failure (binary not found, permission, etc.)
				return exitcodes.Wrap(exitcodes.ChildSpawnFailed, err)
			}
			return nil
		},
	}

	// Profile selectors
	cmd.Flags().StringVar(&awsProf, "aws", "", "AWS profile to inject")
	cmd.Flags().StringVar(&pgProf, "pg", "", "Postgres profile to inject")
	cmd.Flags().StringVar(&mysqlProf, "mysql", "", "MySQL profile to inject")
	cmd.Flags().StringVar(&ghProf, "github", "", "GitHub profile to inject")

	// Env behavior
	cmd.Flags().BoolVar(&noInherit, "no-inherit", false, "do not inherit current environment")

	return cmd
}
