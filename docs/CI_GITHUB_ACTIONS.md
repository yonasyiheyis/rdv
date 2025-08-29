# GitHub Actions — rdv recipes


## Option A: Use the published Docker image (simplest)
```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    container: ghcr.io/yonasyiheyis/rdv:latest
    steps:
      - uses: actions/checkout@v4

      # Write merged env to $GITHUB_ENV (for later steps)
      - name: Inject env via rdv (JSON -> $GITHUB_ENV)
        run: |
          rdv env export \
            --set aws:dev \
            --set db.postgres:dev \
            --set github:bot \
            --json | jq -r 'to_entries[] | "\(.key)=\(.value)"' >> "$GITHUB_ENV"

      - name: Run tests with env
        run: go test ./...
```

## Option B: Install rdv on the runner
```yaml
- name: Install rdv
  run: |
    curl -L -o rdv.tgz https://github.com/yonasyiheyis/rdv/releases/download/${{ env.RDV_TAG }}/rdv_${{ env.RDV_TAG#v }}_linux_amd64.tar.gz
    tar -xzf rdv.tgz
    sudo mv rdv /usr/local/bin/rdv
    rdv --version
```

**Use rdv exec**
```yaml
- name: Run integration tests (AWS + PG)
  run: rdv exec --aws dev --pg dev -- go test -v ./integration/...
```

**Handle failures via exit codes**
```yaml
- name: Validate GitHub token
  run: rdv github set-config --profile ci --no-prompt --token "$TOKEN" --test-conn
  env:
    TOKEN: ${{ secrets.CI_GITHUB_TOKEN }}
# If invalid, step exits with appropriate code and the job fails.
```
