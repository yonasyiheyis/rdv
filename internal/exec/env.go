package execenv

import (
	"maps"
	"os"
	"strings"

	aws "github.com/yonasyiheyis/rdv/internal/plugins/aws"
	db "github.com/yonasyiheyis/rdv/internal/plugins/db"
	gh "github.com/yonasyiheyis/rdv/internal/plugins/github"
)

type Options struct {
	AWS       string
	Postgres  string
	MySQL     string
	GitHub    string
	NoInherit bool
}

// BuildEnv composes environment variables for the selected profiles.
// Reuses per-plugin ExportVars helpers.
func BuildEnv(o Options) (map[string]string, error) {
	env := map[string]string{}

	// inherit current process env unless told not to
	if !o.NoInherit {
		for _, kv := range os.Environ() {
			if i := strings.IndexByte(kv, '='); i >= 0 {
				env[kv[:i]] = kv[i+1:]
			}
		}
	}

	// Merge in each selected profile (later ones win on key collisions).
	if o.AWS != "" {
		m, err := aws.ExportVars(o.AWS)
		if err != nil {
			return nil, err
		}
		maps.Copy(env, m)
	}
	if o.Postgres != "" {
		m, err := db.PGExportVars(o.Postgres)
		if err != nil {
			return nil, err
		}
		maps.Copy(env, m)
	}
	if o.MySQL != "" {
		m, err := db.MySQLExportVars(o.MySQL)
		if err != nil {
			return nil, err
		}
		maps.Copy(env, m)
	}
	if o.GitHub != "" {
		m, err := gh.ExportVars(o.GitHub)
		if err != nil {
			return nil, err
		}
		maps.Copy(env, m)
	}

	return env, nil
}
