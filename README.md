# CouponGo

CouponGo is a command-line tool specifically designed for managing Coupons and Promotion Codes in your Stripe account. It supports multi-environment management, batch operations, and provides both table and JSON output formats.

## Features

- 🔧 **Multi-Environment Management**: Support for test, production, and other environments with independent API key configurations
- 💳 **Coupon Management**: Create, view, update, and delete Stripe coupons
- 🎫 **Promotion Code Management**: Create promotion codes for coupons, supporting both single and batch creation
- 🔐 **Secure Configuration**: API keys are securely stored in local configuration file (`~/.coupongo.json`)
- 📊 **Flexible Output**: Support for table, JSON, and detailed list output formats
- ⚡ **Interactive Experience**: Friendly interactive configuration and operation wizards

## Quick Start

### Installation

#### Method 1: Using Make (Recommended)

```bash
# Clone the project
git clone <repository-url>
cd coupongo

# Install to user directory ~/.local/bin (recommended, no sudo required)
make install-user

# Or install to system directory /usr/local/bin (requires sudo)
make install

# Verify installation
coupongo version
```

#### Method 2: Manual Installation

```bash
# Build binary
go build -o coupongo ./cmd/cli

# Create user bin directory (if it doesn't exist)
mkdir -p ~/.local/bin

# Copy to user bin directory
cp coupongo ~/.local/bin/

# Add to PATH (if not already added)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
source ~/.zshrc  # or source ~/.bashrc
```

#### Method 3: Using go install

```bash
cd coupongo
go install ./cmd/cli
# Binary will be installed to $GOPATH/bin or $GOBIN
```

### Initial Configuration

```bash
# First-time setup, initialize configuration
./coupongo config init
```

The system will guide you through:
1. Choose environment name (e.g., test, production)
2. Enter Stripe API key
3. Set default currency (e.g., usd, eur)
4. Select output format (table or json)

## Basic Usage

### Configuration Management

```bash
# View current configuration
./coupongo config show

# List all environments
./coupongo config list-env

# Switch environment
./coupongo config use production

# Add new environment
./coupongo config add-env staging

# Set API key for environment
./coupongo config set-key production
```

### Coupon Management

```bash
# List all coupons
./coupongo coupon list

# View specific coupon
./coupongo coupon get coup_xxxxx

# Create new coupon (interactive)
./coupongo coupon create

# Update coupon
./coupongo coupon update coup_xxxxx

# Delete coupon
./coupongo coupon delete coup_xxxxx
```

### Promotion Code Management

```bash
# List all promotion codes
./coupongo promo list

# List promotion codes for specific coupon
./coupongo promo list --coupon coup_xxxxx

# Create promotion code for coupon
./coupongo promo create coup_xxxxx

# Batch create promotion codes
./coupongo promo batch coup_xxxxx

# Batch create with command line arguments
./coupongo promo batch coup_xxxxx --count 50 --prefix SAVE --max-redemptions 1

# View specific promotion code
./coupongo promo get promo_xxxxx

# Update promotion code status
./coupongo promo update promo_xxxxx
```

### Global Options

All commands support the following global options:

```bash
# Use specific environment (overrides current environment)
./coupongo coupon list --env production

# Different output formats
./coupongo coupon list --format table    # Table format (default)
./coupongo coupon list --format json     # JSON format (with syntax highlighting)
./coupongo coupon list --format list     # List format (detailed information)

# Combined usage
./coupongo promo list --env test --format list --coupon coup_xxxxx
```

### Output Format Guide

#### Table Format (table)
- Concise table display, suitable for quick browsing
- Uses colors and symbols to highlight status information
- Shows only key information

#### JSON Format (json)
- Complete data output, suitable for programmatic processing
- Syntax highlighting for improved readability
- Includes all fields returned by the API

#### List Format (list)
- Detailed list display, suitable for viewing details
- Uses icons and colors for beautification
- Includes more descriptive information

## Configuration File

The configuration file is located at `~/.coupongo.json` with the following format:

```json
{
  "current_environment": "test",
  "environments": {
    "test": {
      "stripe_api_key": "sk_test_xxxxx",
      "default_currency": "usd",
      "output_format": "table"
    },
    "production": {
      "stripe_api_key": "sk_live_xxxxx",
      "default_currency": "usd",
      "output_format": "table"
    }
  }
}
```

## Security Notes

- API keys are securely stored locally with file permissions set to 600 (user read/write only)
- Supports both test keys (`sk_test_`) and live keys (`sk_live_`)
- API keys are automatically masked when displayed in configuration

## Uninstall

### Using Make to Uninstall

```bash
make uninstall
```

### Manual Uninstall

```bash
# Remove binary
rm ~/.local/bin/coupongo  # or sudo rm /usr/local/bin/coupongo

# Remove configuration file (optional)
rm ~/.coupongo.json
```

## Development

### Project Structure

```
coupongo/
├── cmd/cli/           # CLI application entry point
├── internal/
│   ├── cli/          # CLI command implementations
│   ├── config/       # Configuration management
│   └── stripe/       # Stripe API wrappers
├── pkg/types/        # Shared type definitions
└── CLAUDE.md         # Project documentation
```

### Build and Test

```bash
# Build
go build -o coupongo ./cmd/cli

# Run tests
go test ./...

# Format code
go fmt ./...

# Static analysis
go vet ./...
```

## Installation from GitHub

Once released, you can install directly from GitHub:

```bash
# Install latest version
go install github.com/yourusername/coupongo/cmd/cli@latest

# Install specific version
go install github.com/yourusername/coupongo/cmd/cli@v0.1.0
```

## Contributing

Issues and Pull Requests are welcome!

## License

[MIT License](LICENSE)