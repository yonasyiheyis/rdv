# GitLab CI — rdv snippet

```yaml
stages: [test]

variables:
  RDV_IMAGE: "ghcr.io/yonasyiheyis/rdv:latest"

test:
  stage: test
  image: $RDV_IMAGE
  script:
    - rdv env export --set aws:dev --set db.postgres:dev --json \
      | jq -r 'to_entries[] | "\(.key)=\(.value)"' >> .rdv.env
    - set -a && source .rdv.env && set +a
    - go test ./...
```
