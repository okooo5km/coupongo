# Changelog

All notable changes to CouponGo will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-05-25

### Added
- AI mode with structured JSON envelopes, clean stdout/stderr separation, no-color output, and prompt-free execution.
- Machine-readable `schema` command for command discovery, flags, mutation markers, and error kinds.
- `doctor` command for local readiness checks and optional Stripe connectivity checks.
- Non-interactive flags for coupon, promotion-code, and configuration write workflows.
- Built-in Codex Skill at `skills/coupongo/SKILL.md`.
- GoReleaser release pipeline with Homebrew tap publishing to `okooo5km/homebrew-tap`.

### Changed
- JSON output is now plain pretty-printed JSON instead of ANSI-highlighted JSON.
- CI now checks module tidiness, GoReleaser config, static analysis, and cross-platform builds.
- Release builds now inject the tag version into `coupongo version`.

### Fixed
- Stripe SDK logging is silenced so AI-mode stderr remains valid structured JSON.
- `skills/coupongo` is no longer ignored as if it were the local binary.

## [0.1.1] - 2025-09-25

### Changed
- Centralized CLI version handling so builds can inject release metadata.

## [0.1.0] - 2025-08-27

### Added
- Initial CouponGo CLI for Stripe coupon and promotion-code management.
- Multi-environment Stripe API configuration.
- Interactive coupon and promotion-code workflows.
- Table, JSON, and list output formats.
- Cross-platform build and GitHub Actions release scaffolding.
