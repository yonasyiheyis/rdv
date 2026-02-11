# rdv — ReadyDev CLI

[![release](https://img.shields.io/github/v/release/yonasyiheyis/rdv)](https://github.com/yonasyiheyis/rdv/releases)
[![docker](https://img.shields.io/badge/ghcr.io-rdv-blue?logo=docker)](https://github.com/users/yonasyiheyis/packages/container/package/rdv)

ReadyDev CLI is a focused, interactive tool for managing local and CI development secrets and service credentials. It supports AWS, GCP, PostgreSQL, MySQL, and GitHub, with profile-based configuration, JSON output, and safe non-interactive workflows.

## Highlights

- Interactive or non-interactive setup (`--no-prompt`).
- Profile isolation (`--profile dev|staging|prod`).
- Export to shell, `.env`, or JSON.
- Merge multiple profiles into a single output.
- Run commands with injected credentials via `rdv exec`.
- Stable, script-friendly exit codes.
- Extensible plugin architecture.

## Documentation

- Agents: [docs/AGENTS.md](docs/AGENTS.md)
- GitHub Actions: [docs/CI_GITHUB_ACTIONS.md](docs/CI_GITHUB_ACTIONS.md)
- GitLab CI: [docs/CI_GITLAB.md](docs/CI_GITLAB.md)

## Supported Domains

| Domain | Commands | Behavior | Storage | Output |
|---|---|---|---|---|
| AWS | `set-config`, `modify`, `delete`, `export`, `list`, `show` | Interactive or `--no-prompt` | `~/.aws/{credentials,config}` | `AWS_*`, `--env-file`, `--json` |
| GCP | `set-config`, `modify`, `delete`, `export`, `list`, `show`, `test-conn` | Service account JSON or gcloud ADC | `~/.config/rdv/gcp/<profile>.yaml` | `GOOGLE_*` / `CLOUDSDK_*`, `--env-file`, `--json` |
| PostgreSQL | `db postgres set-config`, `modify`, `delete`, `export`, `list`, `show` | Interactive or `--no-prompt` | `~/.config/rdv/db/postgres.yaml` | `PG*`, `PG_DATABASE_URL`, `--env-file`, `--json` |
| MySQL | `db mysql set-config`, `modify`, `delete`, `export`, `list`, `show` | Interactive or `--no-prompt` | `~/.config/rdv/db/mysql.yaml` | `MYSQL_*`, `MYSQL_DATABASE_URL`, `--env-file`, `--json` |
| GitHub | `github set-config`, `modify`, `delete`, `export`, `list`, `show` | Token profiles | `~/.config/rdv/github.yaml` | `GITHUB_TOKEN`, `--env-file`, `--json` |

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap yonasyiheyis/rdv
brew install rdv
```

### Manual (all OSes)

Download the latest binary from GitHub Releases and move it into your `PATH`:

```bash
chmod +x rdv_1.0.0_darwin_arm64/rdv
sudo mv rdv /usr/local/bin/
```

### Docker

```bash
docker run --rm ghcr.io/yonasyiheyis/rdv:latest rdv --help
# mount your .aws or config dirs as needed:
docker run --rm -v $HOME/.aws:/root/.aws ghcr.io/yonasyiheyis/rdv rdv aws export
```

### Windows (Scoop)

```powershell
# once you create a scoop bucket later; for now direct download
curl -LO https://github.com/yonasyiheyis/rdv/releases/download/v1.0.0/rdv_1.0.0_windows_amd64.zip
Expand-Archive rdv_1.0.0_windows_amd64.zip -DestinationPath C:\rdv
setx PATH "%PATH%;C:\rdv"
```

## Quick Start

```bash
# 1. Configure profiles (and verify them)
rdv aws set-config --profile dev --test-conn
rdv gcp set-config --profile dev --test-conn
rdv db mysql set-config --profile dev
rdv github set-config --profile personal

# 2. Load credentials into the current shell
eval "$(rdv aws export --profile dev)"
eval "$(rdv gcp export --profile dev)"

# 3. Modify or delete profiles later
rdv aws modify --profile dev --test-conn
rdv aws delete --profile dev

# 4. Configure local Postgres (and verify it)
rdv db postgres set-config --profile dev --test-conn

# 5. Inject DATABASE_URL for scripts
eval "$(rdv db postgres export --profile dev)"

# 6. Merge multiple profiles into one .env
rdv env export \
  --set aws:dev \
  --set gcp:dev \
  --set db.postgres:dev \
  --set db.mysql:ci \
  --set github:bot \
  --env-file .env.merged

# 7. Run commands with injected env
rdv exec --aws dev --gcp dev --pg dev -- make test
```

## Usage

### Connection Testing (`--test-conn`)

```bash
rdv aws set-config --profile prod --test-conn
rdv gcp set-config --profile prod --test-conn
rdv db postgres modify --profile staging --test-conn
```

### Non-interactive mode (CI and agents)

All `set-config` and `modify` commands support `--no-prompt` with flags.

GCP:

```bash
rdv gcp set-config --profile ci --no-prompt \
  --auth service-account-json --project-id my-project \
  --key-file /path/to/key.json --region us-central1

rdv gcp set-config --profile ci --no-prompt \
  --auth gcloud-adc --project-id my-project --region us-central1

rdv gcp export --profile ci --env-file .env.ci
```

MySQL:

```bash
rdv db mysql set-config --profile ci --no-prompt \
  --host localhost --port 3306 --dbname app --user ci --password 's3cr3t' \
  --params 'parseTime=true'

rdv db mysql export --profile ci --env-file .env.ci
```

GitHub:

```bash
rdv github set-config --profile bot --no-prompt \
  --token ghp_xxx --api-base https://api.github.com/

rdv github export --profile bot --env-file .env.ci
```

### Interactive rendering mode

By default, interactive prompts use an accessible mode to avoid duplicated fields in limited terminals. To enable the full TUI experience, set:

```bash
RDV_TUI=1 rdv aws set-config
```

### Merge profiles (`rdv env export`)

```bash
# Print merged exports (stdout)
rdv env export --set aws:dev --set gcp:dev --set db.postgres:dev --set github:bot

# Write to a .env file (merge/overwrite keys if present)
rdv env export --set aws:dev --set gcp:dev --set db.mysql:ci --env-file .env.ci

# JSON for agents/CI
rdv env export --set aws:dev --set gcp:dev --set db.postgres:dev --json
```

Notes:

- The order of `--set` flags determines precedence when the same key appears multiple times (later wins).
- `--env-file` writes a simple `KEY=VALUE` file and merges if the file already exists.

### Run commands with injected env (`rdv exec`)

```bash
# Separate rdv flags from your command with `--`
rdv exec --aws dev -- env | grep AWS_
rdv exec --gcp dev -- env | grep GOOGLE_

# Mix multiple profiles
rdv exec --aws dev --gcp dev --pg dev -- make test

# Isolate from your current shell env
rdv exec --no-inherit --mysql ci -- /bin/sh -lc 'echo $MYSQL_DATABASE_URL'
```

Notes:

- At least one of `--aws`, `--gcp`, `--pg`, `--mysql`, or `--github` is required.
- By default your current environment is included; add `--no-inherit` to start clean.
- Stdout/stderr/stdin are streamed through and the child process exit code is returned.

### Exit codes

- `0` success
- `2` invalid usage or missing required non-interactive flags
- `3` profile not found
- `5` connection test failed

Examples:

```bash
rdv github export --profile __nope__ ; echo "exit=$?"     # -> 3
rdv db mysql set-config --no-prompt --profile tmp \        # missing --password
  --host h --port 3306 --dbname d --user u ; echo "exit=$?" # -> 2
rdv db mysql set-config --no-prompt --profile bad \
  --host 127.0.0.1 --port 59998 --dbname x --user u --password p \
  --test-conn ; echo "exit=$?"                              # -> 5
```

## Shell Completion

```bash
# Zsh
rdv completion zsh > $(brew --prefix)/share/zsh/site-functions/_rdv
exec zsh

# Bash (macOS)
rdv completion bash > /usr/local/etc/bash_completion.d/rdv
source /usr/local/etc/bash_completion.d/rdv
```

## Configuration Files Written

| File | Created by | Purpose |
|---|---|---|
| `~/.aws/credentials` / `~/.aws/config` | `rdv aws set-config` | Standard AWS SDK files. |
| `~/.config/rdv/gcp/<profile>.yaml` | `rdv gcp set-config` | YAML storing GCP profiles (per-profile files). |
| `~/.config/rdv/db/postgres.yaml` | `rdv db postgres set-config` | YAML storing multiple Postgres profiles. |
| `~/.config/rdv/db/mysql.yaml` | `rdv db mysql set-config` | YAML storing multiple MySQL profiles. |
| `~/.config/rdv/github.yaml` | `rdv github set-config` | YAML storing multiple GitHub token profiles. |

## Contributing

1. Fork the repo and clone it locally.
2. Install tooling: `brew install go golangci-lint`.
3. Run checks: `make lint test build`.
4. Open a PR with a clear description.

We follow Conventional Commits and Semantic Versioning.

## License

MIT © Yonas Yiheyis & contributors. See [LICENSE](LICENSE) for details.
