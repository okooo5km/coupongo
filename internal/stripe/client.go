package stripe

import (
	"fmt"

	"coupongo/internal/config"
	"coupongo/pkg/types"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
)

// Client wraps the Stripe client with environment-aware configuration
type Client struct {
	sc     *client.API
	config *config.Manager
}

// NewClient creates a new Stripe client
func NewClient(configManager *config.Manager) *Client {
	return &Client{
		config: configManager,
	}
}

// Initialize initializes the Stripe client for a specific environment
func (c *Client) Initialize(envName string) error {
	var env *types.Environment
	var err error

	if envName != "" {
		env, err = c.config.GetEnvironment(envName)
	} else {
		env, err = c.config.GetCurrentEnvironmentConfig()
	}

	if err != nil {
		return fmt.Errorf("failed to get environment config: %w", err)
	}

	if env.StripeAPIKey == "" {
		currentEnv := envName
		if currentEnv == "" {
			currentEnv = c.config.GetCurrentEnvironment()
		}
		return fmt.Errorf("no API key found for environment '%s'", currentEnv)
	}

	// Create new Stripe client
	c.sc = &client.API{}
	c.sc.Init(env.StripeAPIKey, nil)

	// Also set the global key for compatibility with older API patterns
	stripe.Key = env.StripeAPIKey

	return nil
}

// GetClient returns the underlying Stripe client
func (c *Client) GetClient() *client.API {
	return c.sc
}

// IsInitialized checks if the client is properly initialized
func (c *Client) IsInitialized() bool {
	return c.sc != nil
}

// TestConnection tests the API connection by making a simple API call
func (c *Client) TestConnection() error {
	if !c.IsInitialized() {
		return fmt.Errorf("client not initialized")
	}

	// Make a simple API call to test connectivity
	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("limit", "", "1")

	iter := c.sc.Customers.List(params)
	// Just try to get the first item or check if there's an error
	for iter.Next() {
		break
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// GetCurrentEnvironment returns the current environment configuration
func (c *Client) GetCurrentEnvironment() (*types.Environment, error) {
	return c.config.GetCurrentEnvironmentConfig()
}
