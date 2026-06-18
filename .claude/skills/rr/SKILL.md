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
  version: "1.2.0"
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

## Pagination

All `list` commands return **max 50 results by default**. Use `--limit` and `--offset` to paginate:

```bash
rr domain list --limit 100              # First 100
rr domain list --limit 100 --offset 100 # Next 100
```

**Loop to get all results:**
```bash
offset=0; while true; do
  batch=$(rr domain list --limit 100 --offset $offset --json)
  [ "$(echo "$batch" | jq length)" -eq 0 ] && break
  echo "$batch" | jq -r '.[].domainName'
  offset=$((offset + 100))
done
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
| `rr domain list` | List domains (paginated, default 50) |
| `rr domain get <domain>` | Get domain details |
| `rr domain check <domain>` | Check availability |
| `rr domain check-bulk <domains...>` | Bulk check (max 50) |
| `rr domain register <domain>` | Register domain |
| `rr domain renew <domain>` | Renew domain |
| `rr domain transfer-in <domain>` | Initiate transfer |

### Contacts
| Command | Description |
|---------|-------------|
| `rr contact list` | List contacts (paginated, default 50) |
| `rr contact create <handle>` | Create contact |
| `rr contact update <handle>` | Update contact |

### DNS Zones
| Command | Description |
|---------|-------------|
| `rr zone list` | List zones (paginated, default 50) |
| `rr zone list --name <domain>` | Filter zones by exact zone name |
| `rr zone list --managed\|--unmanaged` | Filter by managed status |
| `rr zone list --service BASIC\|PREMIUM` | Filter by DNS service tier |
| `rr zone get <id\|domain>` | Get zone with records (ID or domain name) |
| `rr zone get --domain <domain>` | Get zone directly from a domain |
| `rr zone record add <id\|domain>` | Add DNS record (ID or domain name) |
| `rr zone record update <id\|domain>` | Update a DNS record |
| `rr zone record delete <id\|domain>` | Delete a DNS record |
| `rr zone sync <id\|domain> --file records.yaml` | Sync from YAML |

**Zone arguments accept a numeric zone ID _or_ a domain name** — `rr zone get`, `zone update`, `zone delete`, `zone sync`, and all `zone record` subcommands resolve a domain to its zone automatically. No need to look up the numeric ID first.

### Other
| Command | Description |
|---------|-------------|
| `rr status` | Account overview |
| `rr process list` | List processes (paginated, default 50) |
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
# Address the zone by domain name directly — no ID lookup needed:
rr zone record add example.com --type A --name www --content 1.2.3.4 --ttl 3600
rr zone record update example.com --type TXT --name @ \
  --content "v=spf1 include:_spf.example.net ~all" \
  --old-content "v=spf1 ~all"
rr zone get example.com --json | jq -r '.records[]'
```

Use `rr zone list --name`, `--managed`, `--unmanaged`, or `--service` when you need discovery or narrowing.

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
```


## Installation

```bash
brew install dedene/tap/rr
```
