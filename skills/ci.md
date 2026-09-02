# Skill: CI/CD with GitHub Actions

## Workflow: `.github/workflows/ci.yml`

```yaml
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: golangci/golangci-lint-action@v4
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test ./... -race -coverprofile=coverage.out
```

## Release: goreleaser

- Cross-compile: linux/darwin/windows × amd64/arm64
- Changelog auto-generated from conventional commits
- Homebrew tap (optional, phase 7)
