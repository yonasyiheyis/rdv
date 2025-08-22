# **rdv – ReadyDev CLI**

[![release](https://img.shields.io/github/v/release/yonasyiheyis/rdv)](https://github.com/yonasyiheyis/rdv/releases)
[![docker](https://img.shields.io/badge/ghcr.io-rdv-blue?logo=docker)](https://github.com/users/yonasyiheyis/packages/container/package/rdv)

_Unique, interactive, one‑stop CLI for managing local & CI development secrets and service credentials._

---

## ✨ Key Features


| Domain | Commands | What it does |
|---|---|---|
| **AWS** | `set-config`, `modify`, `delete`, `export`, `list`, `show` | Interactive **or** `--no-prompt` with flags; writes **`~/.aws/{credentials,config}`**; prints `export AWS_*` or writes with `--env-file`. |
| **PostgreSQL** | `db postgres set-config / modify / delete / export / list / show` | Interactive **or** `--no-prompt` with flags; stores profiles in **`~/.config/rdv/db/postgres.yaml`**; prints `PG*`/`PG_DATABASE_URL` or writes with `--env-file`. |
| **MySQL** | `db mysql set-config / modify / delete / export / list / show` | Interactive **or** `--no-prompt` with flags; stores profiles in **`~/.config/rdv/db/mysql.yaml`**; prints `MYSQL_*`/`MYSQL_DATABASE_URL` or writes with `--env-file`. |
| **GitHub** | `github set-config / modify / delete / export / list / show` | Manage per-profile tokens; interactive **or** `--no-prompt` with flags; stores in **`~/.config/rdv/github.yaml`**; prints `GITHUB_TOKEN` (and optional vars) or writes with `--env-file`. |
| **Plugin Architecture** | – | Each domain (AWS, DBs, GitHub) is a Go plugin registered at build time—easy to extend. |
| **Profiles** | `--profile dev` | Keep isolated configs (`default`, `dev`, `staging`, …). |
| **Shell-friendly** | `eval "$(rdv … export)"`, `--env-file` | Outputs `export` lines or merges to `.env` files for CI/agents. |
| **Completions** | `rdv completion zsh` | Generates Bash, Zsh, Fish, PowerShell completion scripts. |
| **Structured Logging** | `--debug` | Enable JSON/debug logs powered by zap. |
| **JSON output** | `--json` on `list` / `show` | Stable, sorted output for agents/CI across AWS, Postgres, MySQL, GitHub. |


---

## 📦 Installation

### macOS / Linux (Homebrew)

```bash
brew tap yonasyiheyis/rdv
brew install rdv            # upgrades with `brew upgrade rdv`
```

### Manual (all OSes)

Download the latest binary from the [GitHub Releases page](https://github.com/yonasyiheyis/rdv/releases), then move it into your `$PATH` and make it executable:

```bash
chmod +x rdv_0.6.0_darwin_arm64/rdv
sudo mv rdv /usr/local/bin/
```

### 🚀 Quick Start

```bash
# 1. Configure an AWS profile interactively (and verify it)
rdv aws set-config --profile dev --test-conn
rdv db mysql set-config --profile dev
rdv github set-config --profile personal

# 2. Load the creds into your shell
eval "$(rdv aws export --profile dev)"

# 3. Modify or delete the AWS profile later
rdv aws modify --profile dev --test-conn
rdv aws delete --profile dev

# 4. Configure a local Postgres DB (and verify it)
rdv db postgres set-config --profile dev --test-conn

# 5. Inject DATABASE_URL for test scripts
eval "$(rdv db postgres export --profile dev)"

# 6. Modify/Delete the Postgres profile
rdv db postgres modify --profile dev --test-conn
rdv db postgres delete --profile dev

# 7. List profiles
rdv aws list
rdv db postgres list
rdv db mysql list
rdv github list

# 8. Show a single profile (secrets redacted)
rdv db mysql show --profile ci
rdv github show --profile bot

# 9. Machine-readable (JSON) examples
rdv aws list --json
rdv db postgres show --profile dev --json
```
Tips: 
- add profile‑specific exports to files like .env.dev, .env.test, etc.
- JSON output is available on `list` and `show` for all plugins via `--json`, and lists are sorted for deterministic results.


#### 🧪 Connection Testing (`--test-conn`)

Add `--test-conn` to `set-config` or `modify` to immediately verify credentials:

- **AWS**: calls STS `GetCallerIdentity` to ensure keys/region are valid.
- **PostgreSQL**: opens a connection and pings the database.

Example:

```bash
rdv aws set-config --profile prod --test-conn
rdv db postgres modify --profile staging --test-conn
```

#### 🤖 Non-interactive mode (CI & agents)

Every `set-config` and `modify` supports `--no-prompt` plus flags, so you can configure profiles without TTYs.

**MySQL**
```bash
rdv db mysql set-config --profile ci --no-prompt \
  --host localhost --port 3306 --dbname app --user ci --password 's3cr3t' \
  --params 'parseTime=true'

rdv db mysql modify --profile ci --no-prompt --port 3307
rdv db mysql export --profile ci --env-file .env.ci
```

**GitHub**
```bash
rdv github set-config --profile bot --no-prompt \
  --token ghp_xxx --api-base https://api.github.com/

rdv github export --profile bot --env-file .env.ci
```
(Interactive prompts remain available when --no-prompt is omitted.)

### Docker (Linux/macOS/Windows)

```bash
docker run --rm ghcr.io/yonasyiheyis/rdv:latest rdv --help
# mount your .aws or config dirs as needed:
docker run --rm -v $HOME/.aws:/root/.aws ghcr.io/yonasyiheyis/rdv rdv aws export
```

### Windows (Scoop)

```powershell
# once you create a scoop bucket later; for now direct download
curl -LO https://github.com/yonasyiheyis/rdv/releases/download/v0.6.0/rdv_0.6.0_windows_amd64.zip
Expand-Archive rdv_0.6.0_windows_amd64.zip -DestinationPath C:\rdv
setx PATH "%PATH%;C:\rdv"
```


### 🖥️ Shell Completion

```bash
# Zsh
rdv completion zsh > $(brew --prefix)/share/zsh/site-functions/_rdv
exec zsh

# Bash (macOS)
rdv completion bash > /usr/local/etc/bash_completion.d/rdv
source /usr/local/etc/bash_completion.d/rdv
```

### 🔧 Configuration Files Written

| File                                   | Created by                            | Purpose                                       |
|----------------------------------------|---------------------------------------|-----------------------------------------------|
| `~/.aws/credentials` / `~/.aws/config` | `rdv aws set-config`                  | Standard AWS SDK files.                       |
| `~/.config/rdv/db/postgres.yaml`       | `rdv db postgres set-config`          | YAML storing multiple Postgres profiles.      |
| `~/.config/rdv/db/mysql.yaml`          | `rdv db mysql set-config`             | YAML storing multiple MySQL profiles.         |
| `~/.config/rdv/github.yaml`            | `rdv github set-config`               | YAML storing multiple GitHub token profiles.  |


### 🤝 Contributing

1. Fork the repo and clone it locally.

2. Install tooling:

```bash
brew install go golangci-lint
```

3. Ensure all checks pass before opening a PR:

```bash
make lint test build
```

4. Submit a pull request with a clear description.

We follow Conventional Commits and Semantic Versioning.

### 📄 License
MIT © Yonas Yiheyis & contributors. See [LICENSE](LICENSE) for full details.
