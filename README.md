# 🌐 realtime-register-cli - your domains in the terminal

Command-line interface for the [RealtimeRegister](https://www.realtimeregister.com/) domain
registrar API.

## Agent Skill

This CLI is available as an [open agent skill](https://skills.sh/) for AI assistants including
[Claude Code](https://claude.ai/code), [OpenClaw](https://openclaw.ai/),
[Codex](https://github.com/openai/codex), Cursor, GitHub Copilot, and
[35+ agents](https://github.com/vercel-labs/skills#supported-agents).

```bash
npx skills add dedene/realtime-register-cli
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew install dedene/tap/rr
```

### GitHub Releases

Download from [Releases](https://github.com/dedene/realtime-register-cli/releases).

### Go Install

```bash
go install github.com/dedene/realtime-register-cli/cmd/rr@latest
```

## Quick Start

```bash
# Authenticate
rr auth login

# Set customer handle
rr config set customer mycustomer

# Check domain availability
rr domain check example.com

# List your domains
rr domain list

# Get account status
rr status
```

## Commands

| Command         | Description          |
| --------------- | -------------------- |
| `rr auth`       | Manage API key       |
| `rr config`     | Manage configuration |
| `rr status`     | Show account status  |
| `rr domain`     | Domain management    |
| `rr contact`    | Contact management   |
| `rr zone`       | DNS zone management  |
| `rr process`    | Process tracking     |
| `rr tld`        | TLD information      |
| `rr completion` | Shell completions    |

## Configuration

Config file: `~/.config/rr/config.yaml`

```yaml
customer: mycustomer
default_tlds:
  - com
  - net
  - io
auto_renew: true
```

## Environment Variables

| Variable      | Description                 |
| ------------- | --------------------------- |
| `RR_API_KEY`  | API key (overrides keyring) |
| `RR_CUSTOMER` | Customer handle             |
| `RR_JSON`     | Enable JSON output          |
| `RR_PLAIN`    | Enable TSV output           |
| `NO_COLOR`    | Disable colors              |

## Output Formats

```bash
# Table (default)
rr domain list

# JSON
rr domain list --json

# TSV (for scripting)
rr domain list --plain
```

## Shell Completions

```bash
# Bash
rr completion bash > /etc/bash_completion.d/rr

# Zsh
rr completion zsh > "${fpath[1]}/_rr"

# Fish
rr completion fish > ~/.config/fish/completions/rr.fish
```

## License

MIT
