# CouponGo

CouponGo is an AI-friendly CLI for managing Stripe coupons and promotion codes. It works well for humans in a terminal, shell scripts in CI, and AI agents that need deterministic JSON, stable exit codes, and command introspection.

## Features

- Manage Stripe coupons: list, get, create, update, delete.
- Manage promotion codes: list, get, create, batch create, update active status.
- Use multiple Stripe environments from `~/.coupongo.json`.
- Run safely in automation with `--ai`, `schema`, `doctor`, non-interactive flags, and structured errors.
- Ship an in-repo Codex Skill at `skills/coupongo/SKILL.md`.

## Install

```bash
make build
./build/coupongo version
```

Install for the current user:

```bash
make install-user
coupongo version
```

Install with Homebrew after a tagged release:

```bash
brew install okooo5km/tap/coupongo
```

Install with Go:

```bash
go install ./cmd/cli
```

## First Setup

Interactive setup:

```bash
coupongo config init
```

Non-interactive setup:

```bash
coupongo config init \
  --env-name test \
  --api-key sk_test_xxxxx \
  --currency usd \
  --output-format table \
  --skip-test
```

Configuration is stored at `~/.coupongo.json` with mode `0600`.

## AI-Friendly Contract

Use `--ai` for agent and script workflows:

```bash
coupongo doctor --ai
coupongo schema
coupongo coupon list --ai --env test --limit 20
```

In AI mode:

- Success JSON is written to stdout.
- Error JSON is written to stderr.
- Output uses a stable envelope: `{ "schema_version": 1, "success": true|false, ... }`.
- Colors and prompts are disabled.
- Destructive operations require explicit flags such as `--yes`.

Exit codes:

| Code | Kind | Meaning |
| ---: | --- | --- |
| `0` | success | Command completed successfully |
| `1` | execution | Accepted command failed during execution |
| `64` | usage | Invalid command, flag, argument, or missing non-interactive input |
| `65` | auth | API key or authentication problem |
| `66` | not_found | Environment or Stripe resource was not found |
| `67` | conflict | Requested state conflicts with existing local config |
| `68` | network | Network or Stripe API availability issue |
| `130` | cancelled | Interactive operation was cancelled |

Use `coupongo schema` to inspect commands, flags, mutation markers, and error kinds. Use `coupongo doctor --ai` before automation to check local readiness.

## Global Flags

```bash
--env, -e <name>          Use a configured environment
--format, -f <format>     table | json | list
--output <format>         Alias for --format
--json                    Shortcut for --format json
--ai                      JSON envelope, no color, no prompts, structured errors
--no-color                Disable ANSI color output
```

When stdout is not a terminal and no format is explicitly set, CouponGo defaults to JSON.

## Configuration

```bash
coupongo config show
coupongo config path
coupongo config list-env
coupongo config use production
coupongo config add-env staging --api-key sk_test_xxxxx --currency usd --output-format table
coupongo config set-key staging --api-key sk_test_xxxxx
coupongo config remove-env staging --yes
coupongo config reset --yes
```

Example config:

```json
{
  "current_environment": "test",
  "environments": {
    "test": {
      "stripe_api_key": "sk_test_xxxxx",
      "default_currency": "usd",
      "output_format": "table"
    }
  }
}
```

API keys are masked in `config show`, `doctor`, and JSON output.

## Coupons

```bash
coupongo coupon list --env test --limit 20
coupongo coupon list --env test --limit 20 --starting-after coup_xxxxx
coupongo coupon get coup_xxxxx --env test
```

```bash
coupongo coupon create --env test \
  --percent-off 20 \
  --duration once \
  --name "Launch 20"
```

```bash
coupongo coupon create --env test \
  --amount-off 1500 \
  --currency usd \
  --duration repeating \
  --duration-in-months 3 \
  --max-redemptions 100
```

```bash
coupongo coupon update coup_xxxxx --env test --name "Updated name"
coupongo coupon delete coup_xxxxx --env test --yes
```

Useful create flags:

```bash
--id <id>
--name <text>
--percent-off <number>
--amount-off <integer>
--currency <code>
--duration once|forever|repeating
--duration-in-months <integer>
--max-redemptions <integer>
--redeem-by <unix_timestamp>
--products prod_a,prod_b
--currency-options eur:950,jpy:1500
--metadata key=value
```

## Promotion Codes

```bash
coupongo promo list --env test --limit 50
coupongo promo list --env test --coupon coup_xxxxx --limit 50
coupongo promo get promo_xxxxx --env test
```

Create one exact code:

```bash
coupongo promo create coup_xxxxx --env test \
  --code SAVE20 \
  --max-redemptions 100
```

Create one generated code:

```bash
coupongo promo create coup_xxxxx --env test \
  --prefix SAVE \
  --separator -
```

Batch create:

```bash
coupongo promo batch coup_xxxxx --env test \
  --count 50 \
  --prefix SAVE \
  --max-redemptions 1
```

Update active status:

```bash
coupongo promo update promo_xxxxx --env test --active=false
```

Useful promo flags:

```bash
--code <code>
--prefix <prefix>
--separator -|''
--customer <cus_id>
--active=true|false
--expires-at <unix_timestamp>
--max-redemptions <integer>
--first-time-only
--minimum-amount <integer>
--currency <code>
--metadata key=value
```

## Built-In Skill

CouponGo ships with a Codex Skill:

```text
skills/coupongo/SKILL.md
```

Agents should use it when asked to manage Stripe coupons or promotion codes through CouponGo. The Skill instructs agents to start with `doctor --ai`, inspect `schema`, use non-interactive flags, avoid invented IDs, and require explicit intent before production writes or deletion.

## Development

```bash
make fmt
make vet
make test
make check
make build
```

Project layout:

```text
cmd/cli/           CLI entrypoint
internal/cli/      Cobra commands and output contract
internal/config/   Local environment config
internal/stripe/   Stripe SDK wrappers
pkg/types/         Shared DTOs
skills/coupongo/   Built-in Codex Skill
```

Before release, run:

```bash
make check
goreleaser check
python3 /Users/5km/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/coupongo
```
