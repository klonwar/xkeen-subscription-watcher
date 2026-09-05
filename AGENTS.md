# Repository Guidelines

## Project Structure & Module Organization

This Go CLI module (`module github.com/tkukushkin/xkeen-subscription-watcher`) has a flat layout:

- `main.go` defines the Cobra command, flags, and configuration assembly.
- `subscription.go` downloads and decodes subscription responses.
- `proxy.go` parses VLESS, VMess, Trojan, Shadowsocks, and Hysteria2 URLs.
- `outbound.go` converts parsed proxies into Xray outbound JSON.
- `main_test.go` contains unit and HTTP fixture tests.
- `install.sh` installs the latest architecture-specific release; `.github/workflows/` contains test and release automation.

Generated files are written to the configured output directory as `04_outbounds.<tag>.json`; do not commit generated configs or credentials.

## Build, Test, and Development Commands

Use Go 1.25 (the version declared in `go.mod`). Common commands are:

```sh
go run . --no-restart <tag>=<url>                 # run locally without restarting XKeen
go build -o xkeen-subscription-watcher .          # build the CLI
gofmt -w *.go                                     # format Go sources
go vet ./...                                      # run static checks
go test -v -race -coverprofile=coverage.out ./... # run CI-equivalent tests
```

The test workflow also verifies that `gofmt` reports no changes. Run `go test ./...` while iterating.

## Coding Style & Naming Conventions

Keep code gofmt-formatted and follow standard Go idioms: tabs for indentation, lower-camel unexported names, and UpperCamelCase exported names. Keep protocol parsing in `proxy.go` and outbound serialization in `outbound.go`; avoid duplicating transport/security mapping. Preserve user-facing CLI and log text conventions unless behavior intentionally changes.

## Testing Guidelines

Add table-driven tests in `main_test.go` (or a new `*_test.go` file) for parser or CLI changes. Name tests `Test<Subject>` and helpers descriptively. Cover valid/malformed URLs, empty responses, HTTP failures, flag combinations, and generated JSON. Use `httptest` fixtures, not real network calls, and run the race-enabled command before submitting.

## Commit & Pull Request Guidelines

Existing commits use short, imperative descriptions (often in Russian) without a strict prefix, for example `Добавил поддержку протоколов vmess и trojan`. Keep commits focused and describe user-visible behavior. Pull requests should explain motivation and implementation, list verification commands, note compatibility/configuration changes, and include representative CLI output or generated JSON when output format changes. Do not include subscription URLs, tokens, or machine-specific paths.

## Security & Configuration Tips

Treat subscription URLs and generated Xray configs as sensitive: keep them out of commits, logs, and issues. Review restart behavior, output paths, and proxy fallback carefully; use `--no-restart` during local testing.
