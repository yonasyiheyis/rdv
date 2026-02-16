# rdv — Agent & Automation Contract

This document describes the **machine contract** for AI agents, scripts, and CI.

---

## JSON Shapes

All JSON is printed to stdout when `--json` is present.

### 1) List profiles
```bash
rdv aws list --json
```
```
{ "profiles": ["ci","dev","staging"] }
```

### 2) Show profile (secrets redacted in JSON too)
```
rdv db postgres show --profile dev --json
```
```
{
  "profile": "dev",
  "host": "localhost",
  "port": "5432",
  "dbname": "app",
  "user": "dev",
  "password": "••••••"
}
```

### 3) Export env for one plugin
```
rdv github export --profile bot --json
```
```
{
  "GITHUB_TOKEN": "ghp_***REDACTED***",
  "GITHUB_API_BASE": "https://api.github.com/",
  "GITHUB_USER": "bot-user"
}
```

### 4) Global merge (multiple profiles)
```
rdv env export \
  --set aws:dev --set gcp:dev --set db.postgres:dev --set github:bot --json
```
```
{
  "AWS_ACCESS_KEY_ID":     "AKIA...REDACTED",
  "AWS_SECRET_ACCESS_KEY": "••••••",
  "AWS_DEFAULT_REGION":    "us-east-1",
  "PG_DATABASE_URL":       "postgres://user:••••••@localhost:5432/app?sslmode=disable",
  "PGHOST": "localhost", "PGPORT": "5432", "PGUSER": "user", "PGPASSWORD": "••••••", "PGDATABASE": "app",
  "GITHUB_TOKEN": "ghp_***REDACTED***", "GITHUB_API_BASE": "https://api.github.com/", "GITHUB_USER": "bot-user"
}
```
Precedence: later `--set` wins on key collisions.

### 5) `--env-file` receipts
When writing to a `.env` file, rdv emits a JSON receipt:
```
rdv aws export --profile dev --env-file .env.dev --json
```
```
{ "written": 3, "path": ".env.dev", "vars": { "AWS_ACCESS_KEY_ID": "...", "AWS_SECRET_ACCESS_KEY": "...", "AWS_DEFAULT_REGION": "..." } }
```

### 6) Exit Codes (stable)
`0` — success\
`2` — invalid args / usage error\
`3` — profile not found\
`4` — read/write failure (files on disk)\
`5` — connection error, --test-conn failed (DB host down, etc.)\
`6` — i/o error (cannot read/write config/env files)\
`7` — json output failure

`20` — exec could not start (not found, perms, etc.)\
`1` — unknown failure (default fallback)

*Child process exit codes from `rdv exec` are passed through.

### 7) Exec (ephemeral env injection)
Run any command with env from saved profiles:
```
# separate rdv flags from your command with `--`
rdv exec --aws dev --pg dev -- make test
rdv exec --gcp dev -- env | grep GOOGLE_

# empty baseline (do not inherit current shell env)
rdv exec --no-inherit --mysql ci -- /bin/sh -lc 'echo $MYSQL_DATABASE_URL'
```

### Examples for Agents

**Load env into a process the agent will run**
```
# Fetch merged env JSON, then run a command with those vars set
eval "$(
  rdv env export --set aws:dev --set gcp:dev --set db.postgres:dev \
  --json | jq -r 'to_entries[] | "export \(.key)=\(.value)"'
)"
go test ./...
```

**Use `rdv exec` to run tools directly**
```
rdv exec --aws dev --gcp dev --pg dev -- go test ./...
```

**Discover available profiles**
```
rdv db mysql list --json | jq -r '.profiles[]'
```
