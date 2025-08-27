# Changelog

All notable changes to CouponGo will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of CouponGo
- Multi-environment Stripe API management
- Interactive coupon creation and management
- Promotion code creation (single and batch)
- Three output formats: table, JSON with highlighting, and detailed list
- Secure configuration file management
- Cross-platform binary builds
- GitHub Actions for CI/CD

### Features
- **Coupon Management**: Create, list, view, update, and delete Stripe coupons
- **Promotion Codes**: Create individual or batch promotion codes for coupons
- **Multi-Environment**: Support for test, production, and custom environments
- **Interactive CLI**: User-friendly prompts for all operations
- **Flexible Output**: Table, JSON, and list formats with color coding
- **Secure Storage**: API keys stored securely in `~/.coupongo.json`
- **Cross-Platform**: Binaries available for Linux, macOS, and Windows (AMD64 and ARM64)

### Installation Methods
- Direct `go install` from GitHub
- Download pre-built binaries from releases
- Build from source with provided Makefile

## [0.1.0] - TBD

- Initial release (planned)