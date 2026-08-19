# Contributing to yoke

Thanks for your interest in improving yoke.

## Development

```bash
go build -o yoke ./cmd/yoke   # build
go vet ./...                  # vet
go test ./...                 # test
```

Requires Go (see `go.mod` for the version).

## Workflow

- `main` is protected: changes land through pull requests, not direct pushes.
- Open a PR against `main`; CI (build + vet + test) must pass.
- Keep commits focused and messages descriptive.

## Code style

- Comments explain **why**, not what; skip comments that restate the code.
- Match the style of the surrounding code.
- Add or update tests for behavior changes.

## Reporting issues

Open a GitHub issue with steps to reproduce, expected vs actual behavior, and
your environment (OS, Go version, yoke version from `yoke --version`).
