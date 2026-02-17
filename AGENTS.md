# Repository Guidelines

## Project Structure

- `cmd/rr/`: CLI entrypoint (binary name: `rr`)
- `internal/`: implementation
  - `cmd/`: command routing (kong CLI framework)
  - `api/`: RealtimeRegister API client
  - `auth/`: API key management (keyring)
  - `config/`: YAML config (customer ID, default TLDs, auto-renew, keyring backend)
  - `output/`: formatters (table, JSON, TSV)
  - `errfmt/`: error formatting
- `bin/`: build outputs

**Note**: Embeds timezone data for consistent time handling.

## Build, Test, and Development Commands

- `make build`: compile to `bin/rr`
- `make fmt` / `make lint` / `make test` / `make ci`: format, lint, test, full local gate
- `make tools`: install pinned dev tools into `.tools/`
- `make clean`: remove bin/ and .tools/

## Coding Style & Naming Conventions

- Formatting: `make fmt` (goimports local prefix `github.com/dedene/realtime-register-cli` + gofumpt)
- Output: keep stdout parseable (`--json`, `--plain` for TSV); send human hints/progress to stderr
- Linting: golangci-lint v2.8.0 with project config
- Shell completion supported

## Testing Guidelines

- Unit tests: stdlib `testing` with subtests (`t.Run`)
- 2 test files: DNS record parsing, client mocking
- CI gate: fmt-check, lint, test

## Config & Secrets

- **Keyring**: 99designs/keyring for API key storage
- **Config file**: `~/.config/rr/config.yaml`
  - `customer`: customer ID
  - `default_tlds`: preferred TLDs
  - `auto_renew`: default auto-renewal setting
  - `keyring_backend`: backend preference
- **Env overrides**: `RR_JSON`, `RR_PLAIN` for output format

## Key Commands

- `domain`: list, get, register, renew, transfer domains
- `zone`: DNS zone management (records, list)
- `contact`: contact management
- `process`: async process tracking
- `tld`: TLD information
- Global flags: `--json`, `--plain` (TSV), `--verbose`, `--yes` (skip prompts), `--color`

## Commit & Pull Request Guidelines

- Conventional Commits: `feat|fix|refactor|build|ci|chore|docs|style|perf|test`
- Group related changes; avoid bundling unrelated refactors
- PR review: use `gh pr view` / `gh pr diff`; don't switch branches

## Security Tips

- **Critical**: Never commit API keys (X-API-Key header auth)
- Prefer OS keychain for API key storage
- Domain operations can have financial impact; use `--yes` carefully
