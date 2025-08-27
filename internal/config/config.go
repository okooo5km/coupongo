package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"coupongo/pkg/types"
)

const (
	ConfigFileName = ".coupongo.json"
	ConfigFileMode = 0600 // Read/write for owner only
)

var (
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrInvalidAPIKey       = errors.New("invalid API key format")
)

// Manager handles configuration operations
type Manager struct {
	config   *types.Config
	filePath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	homeDir, _ := os.UserHomeDir()
	filePath := filepath.Join(homeDir, ConfigFileName)

	return &Manager{
		filePath: filePath,
	}
}

// Load loads configuration from file or creates default if not exists
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			m.config = types.DefaultConfig()
			return m.Save()
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if config.Environments == nil {
		config.Environments = make(map[string]types.Environment)
	}

	if config.CurrentEnvironment == "" {
		config.CurrentEnvironment = "test"
	}

	m.config = &config
	return nil
}

// Save saves configuration to file
func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, ConfigFileMode); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentEnvironment returns the current environment name
func (m *Manager) GetCurrentEnvironment() string {
	if m.config == nil {
		return "test"
	}
	return m.config.CurrentEnvironment
}

// GetEnvironment returns environment configuration by name
func (m *Manager) GetEnvironment(name string) (*types.Environment, error) {
	if m.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	env, exists := m.config.Environments[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrEnvironmentNotFound, name)
	}

	return &env, nil
}

// GetCurrentEnvironmentConfig returns current environment configuration
func (m *Manager) GetCurrentEnvironmentConfig() (*types.Environment, error) {
	return m.GetEnvironment(m.GetCurrentEnvironment())
}

// SetCurrentEnvironment sets the current environment
func (m *Manager) SetCurrentEnvironment(name string) error {
	if m.config == nil {
		return fmt.Errorf("config not loaded")
	}

	if _, exists := m.config.Environments[name]; !exists {
		return fmt.Errorf("%w: %s", ErrEnvironmentNotFound, name)
	}

	m.config.CurrentEnvironment = name
	return m.Save()
}

// AddEnvironment adds a new environment
func (m *Manager) AddEnvironment(name string, env types.Environment) error {
	if m.config == nil {
		return fmt.Errorf("config not loaded")
	}

	if name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	// Validate API key format
	if env.StripeAPIKey != "" {
		if err := validateAPIKey(env.StripeAPIKey); err != nil {
			return err
		}
	}

	// Set defaults
	if env.DefaultCurrency == "" {
		env.DefaultCurrency = "usd"
	}
	if env.OutputFormat == "" {
		env.OutputFormat = string(types.OutputFormatTable)
	}

	m.config.Environments[name] = env
	return m.Save()
}

// RemoveEnvironment removes an environment
func (m *Manager) RemoveEnvironment(name string) error {
	if m.config == nil {
		return fmt.Errorf("config not loaded")
	}

	if _, exists := m.config.Environments[name]; !exists {
		return fmt.Errorf("%w: %s", ErrEnvironmentNotFound, name)
	}

	// Cannot remove current environment if it's the last one
	if len(m.config.Environments) == 1 {
		return fmt.Errorf("cannot remove the last environment")
	}

	delete(m.config.Environments, name)

	// If current environment was removed, switch to the first available
	if m.config.CurrentEnvironment == name {
		for envName := range m.config.Environments {
			m.config.CurrentEnvironment = envName
			break
		}
	}

	return m.Save()
}

// UpdateEnvironmentAPIKey updates the API key for an environment
func (m *Manager) UpdateEnvironmentAPIKey(envName, apiKey string) error {
	if m.config == nil {
		return fmt.Errorf("config not loaded")
	}

	env, exists := m.config.Environments[envName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrEnvironmentNotFound, envName)
	}

	if err := validateAPIKey(apiKey); err != nil {
		return err
	}

	env.StripeAPIKey = apiKey
	m.config.Environments[envName] = env
	return m.Save()
}

// ListEnvironments returns all environment names
func (m *Manager) ListEnvironments() []string {
	if m.config == nil {
		return nil
	}

	var envs []string
	for name := range m.config.Environments {
		envs = append(envs, name)
	}
	return envs
}

// GetConfig returns the full configuration
func (m *Manager) GetConfig() *types.Config {
	return m.config
}

// Reset resets configuration to default
func (m *Manager) Reset() error {
	m.config = types.DefaultConfig()
	return m.Save()
}

// validateAPIKey validates the Stripe API key format
func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Stripe API keys start with sk_ (secret keys) or pk_ (publishable keys)
	// We mainly use secret keys for this CLI
	if !strings.HasPrefix(apiKey, "sk_") && !strings.HasPrefix(apiKey, "rk_") {
		return fmt.Errorf("%w: key must start with 'sk_' or 'rk_'", ErrInvalidAPIKey)
	}

	if len(apiKey) < 20 {
		return fmt.Errorf("%w: key too short", ErrInvalidAPIKey)
	}

	return nil
}
