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
	appVersion    = "dev"
)

// SetVersion allows the entrypoint to inject the build version so it stays consistent.
func SetVersion(v string) {
	if v == "" {
		return
	}

	appVersion = v
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Use:           "coupongo",
	Short:         "A CLI tool for managing Stripe coupons and promotion codes",
	Long: "CouponGo is a command line interface for managing Stripe coupons and promotion codes.\n" +
		"It supports multiple environments, batch operations, and provides both table and JSON output formats.\n\n" +
		"Examples:\n" +
		"  coupongo config init                         # Initialize configuration\n" +
		"  coupongo coupon list                         # List all coupons\n" +
		"  coupongo promo batch coupon-1234 --count 50  # Create 50 promotion codes",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		configureRuntime()

		if err := validateOutputFormat(formatFlag); err != nil {
			return err
		}

		// Skip initialization for commands that do not need Stripe API access.
		if cmd.Name() == "version" || cmd.Name() == "schema" || cmd.Name() == "doctor" || isCommandOrParent(cmd, "completion") {
			return nil
		}

		// Config commands manage local state and must not require a usable Stripe key.
		if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
			return nil
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
		targetConfig, err := configManager.GetEnvironment(targetEnv)
		if err != nil {
			if err == config.ErrEnvironmentNotFound {
				return notFoundError(
					fmt.Sprintf("environment %q not found", targetEnv),
					fmt.Sprintf("available environments: %v; run `coupongo config init` or `coupongo config add-env <name>`", configManager.ListEnvironments()),
				)
			}
			return err
		}

		// Ensure API key exists for the environment
		if targetConfig.StripeAPIKey == "" && nonInteractive() {
			return usageError(
				fmt.Sprintf("environment %q has no Stripe API key", targetEnv),
				fmt.Sprintf("run `coupongo config set-key %s --api-key <sk_...>`", targetEnv),
			)
		}
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

func isCommandOrParent(cmd *cobra.Command, name string) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Name() == name {
			return true
		}
	}
	return false
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		renderError(err)
		os.Exit(exitCodeForError(err))
	}
}

func init() {
	// Initialize managers
	configManager = config.NewManager()
	stripeClient = stripe.NewClient(configManager)

	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "", "Environment to use (overrides current environment)")
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "", "Output format (table|json|list)")
	rootCmd.PersistentFlags().StringVar(&formatFlag, "output", "", "Output format alias for --format (table|json|list)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Shortcut for --format json")
	rootCmd.PersistentFlags().BoolVar(&aiFlag, "ai", false, "AI mode: JSON output, no color, no prompts, structured errors")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable ANSI color output")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(couponCmd)
	rootCmd.AddCommand(promoCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		if effectiveOutputFormat("") == FormatJSON {
			_ = renderJSON(map[string]interface{}{
				"version": appVersion,
				"name":    "coupongo",
			})
			return
		}
		fmt.Printf("CouponGo %s\n", appVersion)
		fmt.Println("A CLI tool for managing Stripe coupons and promotion codes")
	},
}
