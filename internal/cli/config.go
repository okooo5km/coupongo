package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"coupongo/pkg/types"

	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage coupongo configuration",
	Long:  "Manage coupongo configuration including environments, API keys, and settings.",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize coupongo configuration",
	Long:  "Initialize coupongo configuration by setting up environments and API keys interactively.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if hasConfigInitFlags(cmd) || !canPrompt() {
			return configInitFromFlags(cmd)
		}

		// Load existing config or create new
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if already configured
		envs := configManager.ListEnvironments()
		if len(envs) > 0 {
			fmt.Printf("Configuration already exists with %d environment(s): %v\n", len(envs), envs)

			prompt := promptui.Select{
				Label: "Do you want to add another environment or reconfigure?",
				Items: []string{"Add new environment", "Reconfigure from scratch", "Cancel"},
			}

			_, choice, err := prompt.Run()
			if err != nil {
				return cancelledError("selection cancelled")
			}

			switch choice {
			case "Add new environment":
				return addEnvironmentInteractive()
			case "Reconfigure from scratch":
				if err := configManager.Reset(); err != nil {
					return fmt.Errorf("failed to reset configuration: %w", err)
				}
				fmt.Println("Configuration reset.")
			case "Cancel":
				return nil
			}
		}

		return configManager.InteractiveSetup()
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Show current configuration including all environments and current settings.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		config := configManager.GetConfig()
		if config == nil {
			fmt.Println("No configuration found. Run 'coupongo config init' to set up.")
			return nil
		}

		if effectiveOutputFormat("") == FormatJSON {
			// Hide API keys in JSON output for security
			configCopy := *config
			configCopy.Environments = make(map[string]types.Environment)
			for name, env := range config.Environments {
				envCopy := env
				if envCopy.StripeAPIKey != "" {
					envCopy.StripeAPIKey = maskAPIKey(envCopy.StripeAPIKey)
				}
				configCopy.Environments[name] = envCopy
			}

			return renderJSON(configCopy)
		} else {
			fmt.Printf("Current Environment: %s\n\n", config.CurrentEnvironment)

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Environment", "API Key", "Currency", "Output Format", "Status"})
			table.SetBorder(false)
			table.SetRowSeparator("-")
			table.SetCenterSeparator("")
			table.SetColumnSeparator(" | ")

			// Sort environments for consistent output
			var envNames []string
			for name := range config.Environments {
				envNames = append(envNames, name)
			}
			sort.Strings(envNames)

			for _, name := range envNames {
				env := config.Environments[name]
				status := "✓"
				if env.StripeAPIKey == "" {
					status = "⚠ No API key"
				}

				current := ""
				if name == config.CurrentEnvironment {
					current = " (current)"
				}

				table.Append([]string{
					name + current,
					maskAPIKey(env.StripeAPIKey),
					env.DefaultCurrency,
					env.OutputFormat,
					status,
				})
			}

			table.Render()
		}

		return nil
	},
}

var configListEnvCmd = &cobra.Command{
	Use:   "list-env",
	Short: "List all environments",
	Long:  "List all configured environments.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		envs := configManager.ListEnvironments()
		if len(envs) == 0 {
			fmt.Println("No environments configured. Run 'coupongo config init' to set up.")
			return nil
		}

		current := configManager.GetCurrentEnvironment()
		sort.Strings(envs)

		if effectiveOutputFormat("") == FormatJSON {
			result := map[string]interface{}{
				"current_environment": current,
				"environments":        envs,
			}
			return renderJSON(result)
		} else {
			fmt.Printf("Current environment: %s\n\n", current)
			fmt.Println("Available environments:")
			for _, env := range envs {
				marker := "  "
				if env == current {
					marker = "* "
				}
				fmt.Printf("%s%s\n", marker, env)
			}
		}

		return nil
	},
}

var configUseCmd = &cobra.Command{
	Use:   "use <environment>",
	Short: "Switch to a different environment",
	Long:  "Switch to a different environment.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		envName := args[0]
		if err := configManager.SetCurrentEnvironment(envName); err != nil {
			return fmt.Errorf("failed to switch environment: %w", err)
		}

		result := map[string]interface{}{
			"current_environment": envName,
		}
		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(result)
		}

		fmt.Printf("Switched to environment: %s\n", envName)
		return nil
	},
}

var configAddEnvCmd = &cobra.Command{
	Use:   "add-env <environment>",
	Short: "Add a new environment",
	Long:  "Add a new environment to the configuration.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		envName := args[0]

		// Check if environment already exists
		if _, err := configManager.GetEnvironment(envName); err == nil {
			return fmt.Errorf("environment '%s' already exists", envName)
		}

		apiKey, _ := cmd.Flags().GetString("api-key")
		if apiKey == "" {
			if !canPrompt() {
				return usageError("config add-env requires --api-key in non-interactive mode", "pass `--api-key <sk_...>`")
			}
			var err error
			apiKey, err = configManager.PromptAPIKey(envName)
			if err != nil {
				return fmt.Errorf("failed to get API key: %w", err)
			}
		}
		currency, _ := cmd.Flags().GetString("currency")
		outputFormat, _ := cmd.Flags().GetString("output-format")
		if err := validateOutputFormat(outputFormat); err != nil {
			return err
		}

		env := types.Environment{
			StripeAPIKey:    apiKey,
			DefaultCurrency: strings.ToLower(currency),
			OutputFormat:    outputFormat,
		}

		if err := configManager.AddEnvironment(envName, env); err != nil {
			return fmt.Errorf("failed to add environment: %w", err)
		}

		result := map[string]interface{}{
			"environment": envName,
			"currency":    env.DefaultCurrency,
			"output":      env.OutputFormat,
		}
		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(result)
		}

		fmt.Printf("Environment '%s' added successfully!\n", envName)
		return nil
	},
}

var configRemoveEnvCmd = &cobra.Command{
	Use:   "remove-env <environment>",
	Short: "Remove an environment",
	Long:  "Remove an environment from the configuration.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		envName := args[0]

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			if !canPrompt() {
				return usageError("config remove-env requires --yes in non-interactive mode", "retry with `--yes` after confirming the environment removal is intended")
			}

			prompt := promptui.Select{
				Label: fmt.Sprintf("Are you sure you want to remove environment '%s'?", envName),
				Items: []string{"Yes", "No"},
			}

			_, choice, err := prompt.Run()
			if err != nil || choice == "No" {
				return cancelledError("operation cancelled")
			}
		}

		if err := configManager.RemoveEnvironment(envName); err != nil {
			return fmt.Errorf("failed to remove environment: %w", err)
		}

		result := map[string]interface{}{
			"removed":     true,
			"environment": envName,
		}
		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(result)
		}

		fmt.Printf("Environment '%s' removed successfully!\n", envName)
		return nil
	},
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key <environment>",
	Short: "Set API key for an environment",
	Long:  "Set or update the API key for a specific environment.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		envName := args[0]

		apiKey, _ := cmd.Flags().GetString("api-key")
		if apiKey == "" {
			if !canPrompt() {
				return usageError("config set-key requires --api-key in non-interactive mode", "pass `--api-key <sk_...>`")
			}
			var err error
			apiKey, err = configManager.PromptAPIKey(envName)
			if err != nil {
				return fmt.Errorf("failed to get API key: %w", err)
			}
		}

		if err := configManager.UpdateEnvironmentAPIKey(envName, apiKey); err != nil {
			return fmt.Errorf("failed to update API key: %w", err)
		}

		result := map[string]interface{}{
			"environment": envName,
			"updated":     true,
		}
		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(result)
		}

		fmt.Printf("API key updated for environment '%s'!\n", envName)
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to default",
	Long:  "Reset configuration to default settings, removing all environments and API keys.",
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			if !canPrompt() {
				return usageError("config reset requires --yes in non-interactive mode", "retry with `--yes` after confirming the reset is intended")
			}

			prompt := promptui.Select{
				Label: "Are you sure you want to reset all configuration? This will remove all environments and API keys.",
				Items: []string{"Yes", "No"},
			}

			_, choice, err := prompt.Run()
			if err != nil || choice == "No" {
				return cancelledError("operation cancelled")
			}
		}

		if err := configManager.Reset(); err != nil {
			return fmt.Errorf("failed to reset configuration: %w", err)
		}

		result := map[string]interface{}{
			"reset": true,
			"path":  configManager.FilePath(),
		}
		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(result)
		}

		fmt.Println("Configuration reset to default!")
		fmt.Println("Run 'coupongo config init' to set up a new configuration.")
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the configuration file path",
	Long:  "Print the absolute path to the coupongo configuration file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		result := map[string]interface{}{
			"path": configManager.FilePath(),
		}
		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(result)
		}

		fmt.Println(configManager.FilePath())
		return nil
	},
}

func init() {
	// Add subcommands to config
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configListEnvCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configAddEnvCmd)
	configCmd.AddCommand(configRemoveEnvCmd)
	configCmd.AddCommand(configSetKeyCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configPathCmd)

	configInitCmd.Flags().String("env-name", "test", "Environment name to create")
	configInitCmd.Flags().String("api-key", "", "Stripe API key for the environment")
	configInitCmd.Flags().String("currency", "usd", "Default currency. ISO 4217 lowercase code")
	configInitCmd.Flags().String("output-format", "table", "Default saved output format. One of: table, json, list")
	configInitCmd.Flags().Bool("skip-test", false, "Skip Stripe API key validation during setup")
	configInitCmd.Flags().Bool("force", false, "Reset existing config before initializing")

	configAddEnvCmd.Flags().String("api-key", "", "Stripe API key for the environment")
	configAddEnvCmd.Flags().String("currency", "usd", "Default currency. ISO 4217 lowercase code")
	configAddEnvCmd.Flags().String("output-format", "table", "Default saved output format. One of: table, json, list")
	configRemoveEnvCmd.Flags().Bool("yes", false, "Confirm removal without an interactive prompt")
	configSetKeyCmd.Flags().String("api-key", "", "Stripe API key for the environment")
	configResetCmd.Flags().Bool("yes", false, "Confirm reset without an interactive prompt")
}

func hasConfigInitFlags(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("env-name") ||
		cmd.Flags().Changed("api-key") ||
		cmd.Flags().Changed("currency") ||
		cmd.Flags().Changed("output-format") ||
		cmd.Flags().Changed("skip-test") ||
		cmd.Flags().Changed("force")
}

func configInitFromFlags(cmd *cobra.Command) error {
	envName, _ := cmd.Flags().GetString("env-name")
	apiKey, _ := cmd.Flags().GetString("api-key")
	currency, _ := cmd.Flags().GetString("currency")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	skipTest, _ := cmd.Flags().GetBool("skip-test")
	force, _ := cmd.Flags().GetBool("force")

	if envName == "" {
		return usageError("config init requires a non-empty --env-name", "pass `--env-name test`")
	}
	if apiKey == "" {
		return usageError("config init requires --api-key in non-interactive mode", "pass `--api-key <sk_...>` or run `coupongo config init` in a terminal")
	}
	if err := validateOutputFormat(outputFormat); err != nil {
		return err
	}

	_, statErr := os.Stat(configManager.FilePath())
	configExists := statErr == nil

	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if force {
		if err := configManager.Reset(); err != nil {
			return fmt.Errorf("failed to reset configuration: %w", err)
		}
	} else if configExists && len(configManager.ListEnvironments()) > 0 {
		return conflictError("configuration already exists", "use `coupongo config add-env <environment> --api-key <sk_...>` or rerun init with `--force`")
	}

	if !skipTest {
		if err := configManager.TestAPIKeyForSetup(apiKey); err != nil {
			return fmt.Errorf("API key test failed: %w", err)
		}
	}

	env := types.Environment{
		StripeAPIKey:    apiKey,
		DefaultCurrency: strings.ToLower(currency),
		OutputFormat:    outputFormat,
	}
	if err := configManager.AddEnvironment(envName, env); err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}
	if err := configManager.SetCurrentEnvironment(envName); err != nil {
		return fmt.Errorf("failed to set current environment: %w", err)
	}

	result := map[string]interface{}{
		"environment": envName,
		"currency":    env.DefaultCurrency,
		"output":      env.OutputFormat,
		"path":        configManager.FilePath(),
	}
	if effectiveOutputFormat("") == FormatJSON {
		return renderJSON(result)
	}

	fmt.Printf("Configuration saved successfully!\n")
	fmt.Printf("   Environment: %s\n", envName)
	fmt.Printf("   Currency: %s\n", env.DefaultCurrency)
	fmt.Printf("   Output: %s\n", env.OutputFormat)
	return nil
}

// addEnvironmentInteractive adds a new environment interactively
func addEnvironmentInteractive() error {
	prompt := promptui.Prompt{
		Label: "New environment name",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("environment name cannot be empty")
			}
			if strings.ContainsAny(input, " \t\n") {
				return fmt.Errorf("environment name cannot contain spaces")
			}
			// Check if exists
			if _, err := configManager.GetEnvironment(input); err == nil {
				return fmt.Errorf("environment '%s' already exists", input)
			}
			return nil
		},
	}

	envName, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("failed to get environment name: %w", err)
	}

	apiKey, err := configManager.PromptAPIKey(envName)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	env := types.Environment{
		StripeAPIKey:    apiKey,
		DefaultCurrency: "usd",
		OutputFormat:    "table",
	}

	if err := configManager.AddEnvironment(envName, env); err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}

	fmt.Printf("✅ Environment '%s' added successfully!\n", envName)
	return nil
}

// maskAPIKey masks an API key for display purposes
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "Not set"
	}

	if len(apiKey) <= 10 {
		return "****"
	}

	prefix := apiKey[:3]             // Show sk_ or rk_
	suffix := apiKey[len(apiKey)-4:] // Show last 4 characters

	return fmt.Sprintf("%s****%s", prefix, suffix)
}
