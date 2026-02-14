---
name: rr-domain-cli
description: |
  Manage domains, DNS zones, and contacts via RealtimeRegister API.
  Use when: checking domain availability, registering/renewing domains,
  managing DNS records, creating contacts, tracking async processes.
  Trigger phrases: "domain availability", "register domain", "DNS records",
  "RealtimeRegister", "rr command", "domain expiry", "transfer domain"
license: MIT
homepage: https://github.com/dedene/realtime-register-cli
metadata:
  author: dedene
  version: "1.1.0"
  openclaw:
    primaryEnv: RR_API_KEY
    requires:
      env:
        - RR_API_KEY
      bins:
        - rr
    install:
      - kind: brew
        tap: dedene/tap
        formula: rr
        bins: [rr]
      - kind: go
        package: github.com/dedene/realtime-register-cli/cmd/rr
        bins: [rr]
---

# rr CLI - RealtimeRegister Domain Management

CLI for RealtimeRegister domain registrar. Manage domains, DNS, contacts, and processes.

## When to Use

- Check domain availability (single or bulk)
- Register, renew, transfer, or delete domains
- Manage DNS zones and records
- Create/update WHOIS contacts
- Monitor async processes (registrations, transfers)
- Check account status and expiring domains

## Prerequisites

```bash
# Install
brew install dedene/tap/rr

# Authenticate (stores in keyring)
rr auth login

# Or use environment variable
export RR_API_KEY=your-api-key

# Set customer handle (required for contacts)
rr config set customer mycustomer
```

## Output Formats

Always use `--json` for parsing. TSV (`--plain`) for simple scripting.

```bash
rr domain list          # Table (human-readable)
rr domain list --json   # JSON (for parsing)
rr domain list --plain  # TSV (tab-separated)
```

## Command Quick Reference

### Domains
| Command | Description |
|---------|-------------|
| `rr domain list` | List all domains |
| `rr domain get <domain>` | Get domain details |
| `rr domain check <domain>` | Check availability |
| `rr domain check-bulk <domains...>` | Bulk check (max 50) |
| `rr domain register <domain>` | Register domain |
| `rr domain renew <domain>` | Renew domain |
| `rr domain transfer-in <domain>` | Initiate transfer |

### Contacts
| Command | Description |
|---------|-------------|
| `rr contact list` | List contacts |
| `rr contact create <handle>` | Create contact |
| `rr contact update <handle>` | Update contact |

### DNS Zones
| Command | Description |
|---------|-------------|
| `rr zone list` | List zones |
| `rr zone get <id>` | Get zone with records |
| `rr zone record add <zoneID>` | Add DNS record |
| `rr zone sync <id> --file records.yaml` | Sync from YAML |

### Other
| Command | Description |
|---------|-------------|
| `rr status` | Account overview |
| `rr process list` | List async processes |
| `rr tld list` | List available TLDs |

## Common Workflows

### Check and Register Domain
```bash
rr domain check example.com --json
rr domain register example.com --registrant mycontact --period 1 -y
```

### Bulk Availability Check
```bash
rr domain check-bulk domain1.com domain2.net domain3.io --json
```

### Create Contact First
```bash
rr contact create myhandle \
  --name "John Doe" --email john@example.com \
  --phone "+1.5551234567" --country US
```

### DNS Zone Management
```bash
ZONE_ID=$(rr zone list --json | jq -r '.[] | select(.name=="example.com") | .id')
rr zone record add $ZONE_ID --type A --name www --content 1.2.3.4 --ttl 3600
```

### Monitor Expiring Domains
```bash
rr domain list --expiring-within 30 --json | jq '.[].domainName'
```

## Parsing JSON Output

```bash
# Get all domain names
rr domain list --json | jq -r '.[].domainName'

# Check if available
rr domain check example.com --json | jq -r '.available'

# Filter by status
rr domain list --json | jq '.[] | select(.status=="active")'
```

## Error Handling

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 3 | Authentication error |
| 4 | API error |

**Common fixes:**
- `not authenticated` → `rr auth login`
- `customer not configured` → `rr config set customer <handle>`
- `rate limited` → Wait and retry, or use bulk endpoints

## Environment Variables

| Variable | Description |
|----------|-------------|
| `RR_API_KEY` | API key (overrides keyring) |
| `RR_CUSTOMER` | Customer handle |
| `RR_JSON` | Enable JSON output |

## Scripting Tips

```bash
# Skip confirmation prompts
rr domain delete example.com -y

# Pagination
rr domain list --limit 100 --offset 0 --json
```


## Installation

```bash
brew install dedene/tap/rr
```
