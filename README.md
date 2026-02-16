# rdv (ReadyDev CLI)

[![release](https://img.shields.io/github/v/release/yonasyiheyis/rdv)](https://github.com/yonasyiheyis/rdv/releases)
[![docker](https://img.shields.io/badge/ghcr.io-rdv-blue?logo=docker)](https://github.com/users/yonasyiheyis/packages/container/package/rdv)

`rdv` centralizes environment configuration for local development, CI/CD pipelines, and coding agents. It provides a single CLI for configuring credentials, switching profiles quickly, and exporting environment variables in flexible formats. It is intended for non-production environments where teams need fast, consistent setup and automation.

## Why rdv
- Local dev: reduce setup friction with a consistent, profile-based configuration flow.
- Fast switching: move between `dev`, `staging`, and other profiles without reconfiguring.
- CI/CD: keep pipeline environment management centralized and script-friendly.
- Agents: let coding agents discover and manage the current environment via one command.
- Extensible: plugins follow a consistent pattern, making new integrations easy to add.

## Features
- Supported plugins: AWS, GCP, PostgreSQL, MySQL, GitHub.
- Interactive and non-interactive configuration (`--no-prompt`).
- Profile management (`default`, `dev`, `staging`, etc.) with fast switching.
- Shell-friendly exports to stdout, `.env` files, or JSON.
- Environment merges across profiles (`rdv env export --set ...`).
- `rdv exec` to run any command with injected credentials and optional env isolation.
- Deterministic JSON output and stable exit codes for CI.
- Plugin architecture with a consistent implementation pattern.

## Installation

### Homebrew (macOS/Linux)
```bash
brew tap yonasyiheyis/rdv
brew install rdv
```

### Manual (all platforms)
Download the latest release from GitHub and place the binary on your `PATH`.

### Docker
```bash
docker run --rm ghcr.io/yonasyiheyis/rdv:latest rdv --help
```

## Quick Start

```bash
# Configure profiles
rdv aws set-config --profile dev --test-conn
rdv gcp set-config --profile dev --test-conn
rdv db postgres set-config --profile dev
rdv github set-config --profile personal

# Load credentials into the current shell
eval "$(rdv aws export --profile dev)"
eval "$(rdv gcp export --profile dev)"

# Merge multiple profiles into a single .env
rdv env export \
  --set aws:dev \
  --set gcp:dev \
  --set db.postgres:dev \
  --set github:bot \
  --env-file .env

# Run a command with injected environment
rdv exec --aws dev --gcp dev --pg dev -- make test
```

## Usage Highlights

`rdv exec` is the primary way agents and CI run commands with a known, centralized environment without manual exports.

### Agent workflow (example)
```bash
# Inspect available profiles (machine-readable)
rdv aws list --json
rdv gcp list --json

# Merge a clean environment for tools or agents
rdv env export --set aws:dev --set gcp:dev --set db.postgres:dev --json

# Run a task with injected credentials
rdv exec --aws dev --gcp dev --pg dev -- go test ./...

# Isolate from the current shell when needed
rdv exec --no-inherit --aws dev -- env | rg '^AWS_'
```

### Export formats
```bash
rdv aws export --profile dev
rdv aws export --profile dev --env-file .env.dev
rdv aws export --profile dev --json
```

### Non-interactive configuration
```bash
rdv gcp set-config --profile ci --no-prompt \
  --auth service-account-json --project-id my-project \
  --key-file /path/to/key.json --region us-central1
```

### `rdv exec`
```bash
rdv exec --no-inherit --mysql ci -- /bin/sh -lc 'echo $MYSQL_DATABASE_URL'
rdv exec --aws dev --gcp dev -- env | grep AWS_
rdv exec --aws dev --pg dev -- make test
```

## Configuration Locations
- AWS: `~/.aws/credentials`, `~/.aws/config`
- GCP: `~/.config/rdv/gcp/<profile>.yaml`
- PostgreSQL: `~/.config/rdv/db/postgres.yaml`
- MySQL: `~/.config/rdv/db/mysql.yaml`
- GitHub: `~/.config/rdv/github.yaml`

## Docs
- Agents and CI: `docs/AGENTS.md`
- GitHub Actions: `docs/CI_GITHUB_ACTIONS.md`
- GitLab CI: `docs/CI_GITLAB.md`

## Development

```bash
make build    # build bin/rdv
make test     # run unit tests
make test-ci  # tests + race + coverage
make lint     # golangci-lint
```

Go toolchain target: `1.26.0`.

## Contributing
- Keep commits short and descriptive.
- Include a brief PR summary and the tests you ran.
- For user-facing changes, include a small example output when helpful.

## License
MIT © Yonas Yiheyis & contributors. See `LICENSE`.
