# **rdv – ReadyDev CLI**

_Unique, interactive, one‑stop CLI for managing local & CI development secrets and service credentials._

---

## ✨ Key Features

| Domain | Commands | What it does |
|--------|----------|--------------|
| **AWS** | `set-config`, `export`, `--profile` | Secure, TTY‑based prompt for `AWS_ACCESS_KEY_ID / SECRET_ACCESS_KEY / region`; writes to **`~/.aws/{credentials,config}`** and prints `export AWS_*` lines on demand. |
| **PostgreSQL** | `db postgres set-config / export` | Prompts for host, port, db name, user, password; stores per‑profile YAML under **`~/.config/rdv/db/postgres.yaml`** and generates `DATABASE_URL` or individual `PG*` env vars. |
| **Plugin Architecture** | – | Each domain (AWS, DB, future GitHub, Stripe, etc.) is a Go plugin registered at build time—easy to extend. |
| **Profiles** | `--profile dev` | Keep isolated configs (`default`, `dev`, `staging`, …). |
| **Shell‑friendly** | `eval "$(rdv aws export)"` | Outputs `export` lines—source them or dump to `.env`. |
| **Completions** | `rdv completion zsh` | Generates Bash, Zsh, Fish, PowerShell completion scripts. |
| **Structured Logging** | `--debug` | Enable JSON/debug logs powered by zap. |

---

## 📦 Installation

### macOS / Linux (Homebrew)

```bash
brew tap yonasyiheyis/rdv
brew install rdv            # upgrades with `brew upgrade rdv`

### Manual (all OSes)

Download the latest binary from the [GitHub Releases page](https://github.com/yonasyiheyis/rdv/releases), then move it into your `$PATH` and make it executable:

```bash
chmod +x rdv_0.1.4_darwin_arm64/rdv
sudo mv rdv /usr/local/bin/

### 🚀 Quick Start

```bash
# 1. Configure an AWS profile interactively
rdv aws set-config --profile dev

# 2. Load the creds into your shell
eval "$(rdv aws export --profile dev)"

# 3. Configure a local Postgres DB
rdv db postgres set-config --profile dev

# 4. Inject DATABASE_URL for test scripts
eval "$(rdv db postgres export --profile dev)"

Tip: add profile‑specific exports to files like .env.dev, .env.test, etc.

### 🖥️ Shell Completion

```bash
# Zsh
rdv completion zsh > $(brew --prefix)/share/zsh/site-functions/_rdv
exec zsh

# Bash (macOS)
rdv completion bash > /usr/local/etc/bash_completion.d/rdv
source /usr/local/etc/bash_completion.d/rdv

### 🤝 Contributing

1. Fork the repo and clone it locally.

2. Install tooling:

```bash
brew install go golangci-lint

3. Ensure all checks pass before opening a PR:

```bash
make lint test build

4. Submit a pull request with a clear description.

We follow Conventional Commits and Semantic Versioning.

### 📄 License
MIT © Yonas Yiheyis & contributors. See [LICENSE](LICENSE) for full details.