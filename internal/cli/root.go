package cli

import (
	"fmt"
	"os"

	"coupongo/internal/config"
	"coupongo/internal/stripe"

	"github.com/spf13/cobra"
)

var (
	configManager *config.Manager
	stripeClient  *stripe.Client
	envFlag       string
	formatFlag    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "coupongo",
	Short: "A CLI tool for managing Stripe coupons and promotion codes",
	Long: `CouponGo is a command line interface for managing Stripe coupons and promotion codes.
It supports multiple environments, batch operations, and provides both table and JSON output formats.

Examples:
  coupongo config init                    # Initialize configuration
  coupongo coupon list                    # List all coupons
  coupongo promo batch coup_xxxx --count 50  # Create 50 promotion codes`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for version command
		if cmd.Name() == "version" {
			return nil
		}

		// Skip initialization for config commands that don't need Stripe API
		if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
			if cmd.Name() == "init" || cmd.Name() == "show" || cmd.Name() == "list-env" || cmd.Name() == "reset" {
				return nil
			}
		}

		// Initialize configuration
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Determine which environment to use
		targetEnv := envFlag
		if targetEnv == "" {
			targetEnv = configManager.GetCurrentEnvironment()
		}

		// Check if environment exists
		_, err := configManager.GetEnvironment(targetEnv)
		if err != nil {
			if err == config.ErrEnvironmentNotFound {
				fmt.Printf("Environment '%s' not found.\n", targetEnv)
				fmt.Printf("Available environments: %v\n", configManager.ListEnvironments())
				fmt.Println("\nRun 'coupongo config init' to set up a new environment.")
				return fmt.Errorf("environment not found")
			}
			return err
		}

		// Ensure API key exists for the environment
		if err := configManager.EnsureAPIKey(targetEnv); err != nil {
			return fmt.Errorf("failed to ensure API key: %w", err)
		}

		// Initialize Stripe client
		if err := stripeClient.Initialize(targetEnv); err != nil {
			return fmt.Errorf("failed to initialize Stripe client: %w", err)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Initialize managers
	configManager = config.NewManager()
	stripeClient = stripe.NewClient(configManager)

	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "", "Environment to use (overrides current environment)")
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "", "Output format (table|json|list)")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(couponCmd)
	rootCmd.AddCommand(promoCmd)
	rootCmd.AddCommand(versionCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CouponGo v1.0.0")
		fmt.Println("A CLI tool for managing Stripe coupons and promotion codes")
	},
}
