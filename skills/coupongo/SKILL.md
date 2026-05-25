---
name: coupongo
description: Use this skill when Codex needs to operate CouponGo, the AI-friendly CLI for managing Stripe coupons and promotion codes. Trigger for tasks involving `coupongo`, Stripe coupon or promotion-code listing, creation, update, deletion, batch generation, configuration inspection, `coupongo schema`, `coupongo doctor`, or agent-safe non-interactive CouponGo workflows.
---

# CouponGo

## Workflow

1. Resolve the CLI:
   - In this repository, run `make build` and use `./build/coupongo`.
   - Outside the repository, use `coupongo` from `PATH`.
2. Inspect readiness with `coupongo doctor --ai`.
3. Inspect the current command contract with `coupongo schema`.
4. For agent-run operations, prefer `--ai --env <environment>` and parse the JSON envelope.
5. For list commands, pass `--limit <1..100>` and use `--starting-after <id>` for pagination.

## Rules

- Use non-interactive flags. Do not rely on prompts.
- Do not invent Stripe IDs. List or get resources first, then act on exact IDs.
- Treat `--ai` as the stable automation contract: JSON on stdout for success, JSON on stderr for errors, no ANSI color, no prompts.
- Check `error.kind` before retrying. Fix `usage` locally; ask for config or credentials on `auth`; list resources again on `not_found`; retry only `network`.
- Do not run production writes unless the user explicitly requests production/live or confirms the target environment.
- For destructive coupon deletion, use `--yes` only after user intent is explicit.
- Never expose real Stripe API keys. Use masked values from `doctor` or `config show --ai`.

## Common Commands

```bash
coupongo doctor --ai
coupongo schema
coupongo config path --ai
coupongo config list-env --ai
```

```bash
coupongo coupon list --ai --env test --limit 20
coupongo coupon get <coupon_id> --ai --env test
coupongo coupon create --ai --env test --percent-off 20 --duration once --name "Launch 20"
coupongo coupon create --ai --env test --amount-off 1500 --currency usd --duration repeating --duration-in-months 3
coupongo coupon update <coupon_id> --ai --env test --name "Updated name"
coupongo coupon delete <coupon_id> --ai --env test --yes
```

```bash
coupongo promo list --ai --env test --coupon <coupon_id> --limit 50
coupongo promo get <promo_id> --ai --env test
coupongo promo create <coupon_id> --ai --env test --code SAVE20 --max-redemptions 100
coupongo promo create <coupon_id> --ai --env test --prefix SAVE --separator -
coupongo promo batch <coupon_id> --ai --env test --count 50 --prefix SAVE --max-redemptions 1
coupongo promo update <promo_id> --ai --env test --active=false
```

## Configuration

Use config writes only when the user provides the key or asks to configure CouponGo:

```bash
coupongo config init --ai --env-name test --api-key <sk_...> --currency usd --output-format table --skip-test
coupongo config add-env staging --ai --api-key <sk_...> --currency usd --output-format table
coupongo config set-key staging --ai --api-key <sk_...>
coupongo config use staging --ai
coupongo config remove-env staging --ai --yes
```
