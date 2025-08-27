package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"coupongo/pkg/types"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
)

// InteractiveSetup guides user through initial configuration setup
func (m *Manager) InteractiveSetup() error {
	fmt.Println("Welcome to CouponGo! Let's set up your configuration.")

	// Ask for environment name
	envPrompt := promptui.Prompt{
		Label:   "Environment name (e.g., test, production, dev)",
		Default: "test",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("environment name cannot be empty")
			}
			if strings.ContainsAny(input, " \t\n") {
				return fmt.Errorf("environment name cannot contain spaces")
			}
			return nil
		},
	}

	envName, err := envPrompt.Run()
	if err != nil {
		return fmt.Errorf("failed to get environment name: %w", err)
	}

	// Ask for API key using bufio to handle long inputs properly
	fmt.Print("Stripe API Key (starts with sk_): ")
	reader := bufio.NewReader(os.Stdin)
	apiKeyInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey := strings.TrimSpace(apiKeyInput)

	// Validate the API key
	if err := validateAPIKey(apiKey); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	// Ask for default currency
	currencyPrompt := promptui.Prompt{
		Label:   "Default currency",
		Default: "usd",
		Validate: func(input string) error {
			if len(input) != 3 {
				return fmt.Errorf("currency should be 3 letters (e.g., usd, eur)")
			}
			return nil
		},
	}

	currency, err := currencyPrompt.Run()
	if err != nil {
		return fmt.Errorf("failed to get currency: %w", err)
	}

	// Ask for output format
	formatSelect := promptui.Select{
		Label: "Default output format",
		Items: []string{"table", "json"},
	}

	_, format, err := formatSelect.Run()
	if err != nil {
		return fmt.Errorf("failed to get output format: %w", err)
	}

	// Test API key
	fmt.Println("Testing API key...")
	if err := m.testAPIKey(apiKey); err != nil {
		fmt.Printf("Warning: API key test failed: %v\n", err)

		continuePrompt := promptui.Select{
			Label: "Continue anyway?",
			Items: []string{"Yes", "No"},
		}

		_, continueChoice, err := continuePrompt.Run()
		if err != nil || continueChoice == "No" {
			return fmt.Errorf("setup cancelled")
		}
	} else {
		fmt.Println("✅ API key is valid!")
	}

	// Save configuration
	env := types.Environment{
		StripeAPIKey:    apiKey,
		DefaultCurrency: strings.ToLower(currency),
		OutputFormat:    format,
	}

	if err := m.AddEnvironment(envName, env); err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}

	if err := m.SetCurrentEnvironment(envName); err != nil {
		return fmt.Errorf("failed to set current environment: %w", err)
	}

	fmt.Printf("✅ Configuration saved successfully!\n")
	fmt.Printf("   Environment: %s\n", envName)
	fmt.Printf("   Currency: %s\n", currency)
	fmt.Printf("   Output: %s\n", format)

	return nil
}

// PromptAPIKey prompts user for API key for a specific environment
func (m *Manager) PromptAPIKey(envName string) (string, error) {
	fmt.Printf("Enter Stripe API Key for environment '%s' (starts with sk_): ", envName)
	reader := bufio.NewReader(os.Stdin)
	apiKeyInput, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey := strings.TrimSpace(apiKeyInput)

	// Validate the API key
	if err := validateAPIKey(apiKey); err != nil {
		return "", fmt.Errorf("invalid API key: %w", err)
	}

	return apiKey, nil
}

// testAPIKey tests if the API key is valid by making a simple API call
func (m *Manager) testAPIKey(apiKey string) error {
	// Set the API key temporarily
	oldKey := stripe.Key
	stripe.Key = apiKey
	defer func() {
		stripe.Key = oldKey
	}()

	// Make a simple API call to test the key
	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("limit", "", "1")

	iter := customer.List(params)
	// Just try to get the first item or check if there's an error
	for iter.Next() {
		break
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("API key test failed: %w", err)
	}

	return nil
}

// EnsureAPIKey ensures an API key exists for the given environment
// If not found, prompts user to enter it
func (m *Manager) EnsureAPIKey(envName string) error {
	env, err := m.GetEnvironment(envName)
	if err != nil {
		return err
	}

	if env.StripeAPIKey == "" {
		fmt.Printf("No API key found for environment '%s'.\n", envName)

		apiKey, err := m.PromptAPIKey(envName)
		if err != nil {
			return fmt.Errorf("failed to get API key: %w", err)
		}

		if err := m.UpdateEnvironmentAPIKey(envName, apiKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}

		fmt.Println("✅ API key saved successfully!")
	}

	return nil
}
