package types

// Environment represents a Stripe environment configuration
type Environment struct {
	StripeAPIKey    string `json:"stripe_api_key"`
	DefaultCurrency string `json:"default_currency"`
	OutputFormat    string `json:"output_format"`
}

// Config represents the application configuration
type Config struct {
	CurrentEnvironment string                 `json:"current_environment"`
	Environments       map[string]Environment `json:"environments"`
}

// OutputFormat defines supported output formats
type OutputFormat string

const (
	OutputFormatTable OutputFormat = "table"
	OutputFormatJSON  OutputFormat = "json"
)

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		CurrentEnvironment: "test",
		Environments: map[string]Environment{
			"test": {
				StripeAPIKey:    "",
				DefaultCurrency: "usd",
				OutputFormat:    string(OutputFormatTable),
			},
		},
	}
}
