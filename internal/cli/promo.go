package cli

import (
	"fmt"
	"strconv"
	"strings"

	"coupongo/internal/stripe"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// promoCmd represents the promotion code command
var promoCmd = &cobra.Command{
	Use:   "promo",
	Short: "Manage Stripe promotion codes",
	Long:  "Create, list, update, and manage Stripe promotion codes for existing coupons.",
}

var promoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List promotion codes",
	Long:  "List all promotion codes, optionally filtered by coupon.",
	RunE: func(cmd *cobra.Command, args []string) error {
		couponID, _ := cmd.Flags().GetString("coupon")

		promoService := stripe.NewPromotionCodeService(stripeClient)
		codes, err := promoService.ListPromotionCodes(couponID)
		if err != nil {
			return fmt.Errorf("failed to list promotion codes: %w", err)
		}

		if len(codes) == 0 {
			if couponID != "" {
				fmt.Printf("No promotion codes found for coupon: %s\n", couponID)
			} else {
				fmt.Println("No promotion codes found.")
			}
			return nil
		}

		// Determine output format
		format := formatFlag
		if format == "" {
			env, _ := stripeClient.GetCurrentEnvironment()
			if env != nil {
				format = env.OutputFormat
			} else {
				format = "table"
			}
		}

		renderer := NewOutputRenderer(format)
		return renderer.RenderPromotionCodes(codes)
	},
}

var promoGetCmd = &cobra.Command{
	Use:   "get <promo_id>",
	Short: "Get a specific promotion code",
	Long:  "Get details of a specific promotion code by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promoID := args[0]
		promoService := stripe.NewPromotionCodeService(stripeClient)

		code, err := promoService.GetPromotionCode(promoID)
		if err != nil {
			return fmt.Errorf("failed to get promotion code: %w", err)
		}

		// Determine output format
		format := formatFlag
		if format == "" {
			env, _ := stripeClient.GetCurrentEnvironment()
			if env != nil {
				format = env.OutputFormat
			} else {
				format = "table"
			}
		}

		renderer := NewOutputRenderer(format)
		return renderer.RenderPromotionCode(code)
	},
}

var promoCreateCmd = &cobra.Command{
	Use:   "create <coupon_id>",
	Short: "Create a promotion code",
	Long: `Create a promotion code for an existing coupon.

Use flags for quick creation or no flags for interactive prompts.

Available flags:
  --prefix, -p           Prefix for auto-generated code (e.g., BEAR -> BEAR_XXXXXXXX)
  --customer             Restrict to specific customer ID
  --active, -a           Set as active (default: true)
  --expires-at           Expiry timestamp (Unix timestamp)
  --max-redemptions, -m  Maximum redemptions (0 for unlimited)
  --first-time-only      Restrict to first-time transactions only
  --minimum-amount       Minimum amount in cents
  --currency             Currency for minimum amount (default: usd)

Interactive prompts (when no flags used) will guide you through:
  • Promotion code (optional, auto-generated if empty)
  • Customer restriction (optional, specific customer ID)
  • Active status (active or inactive)
  • Expiry timestamp (optional)
  • Maximum redemptions (optional)
  • First-time transaction only (yes/no)
  • Minimum amount restriction (optional, in cents)
  • Currency (for minimum amount)

Examples:
  coupongo promo create coup_1234567890                                    # Interactive creation
  coupongo promo create coup_1234567890 --prefix SAVE                      # Auto-generate with prefix
  coupongo promo create coup_1234567890 --prefix BEAR --max-redemptions 100  # With limits
  coupongo promo create coup_1234567890 --customer cus_xxx --active=false  # Customer-specific, inactive`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		couponID := args[0]

		// Get flags
		prefix, _ := cmd.Flags().GetString("prefix")
		customer, _ := cmd.Flags().GetString("customer")
		active, _ := cmd.Flags().GetBool("active")
		expiresAt, _ := cmd.Flags().GetInt64("expires-at")
		maxRedemptions, _ := cmd.Flags().GetInt64("max-redemptions")
		firstTimeOnly, _ := cmd.Flags().GetBool("first-time-only")
		minimumAmount, _ := cmd.Flags().GetInt64("minimum-amount")
		currency, _ := cmd.Flags().GetString("currency")

		// Verify coupon exists
		couponService := stripe.NewCouponService(stripeClient)
		coupon, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to verify coupon: %w", err)
		}

		fmt.Printf("Creating promotion code for coupon: %s (%s)\n", coupon.ID, stripe.FormatCouponValue(coupon))

		var opts stripe.PromotionCodeCreateOptions

		// Check if any flags were provided
		hasFlags := prefix != "" || customer != "" || expiresAt != 0 || maxRedemptions != 0 ||
			firstTimeOnly || minimumAmount != 0 || cmd.Flags().Changed("active")

		if hasFlags {
			// Use flag values
			opts = stripe.PromotionCodeCreateOptions{
				CouponID: couponID,
				Customer: customer,
				Active:   &active,
			}

			// Generate code with prefix if provided
			if prefix != "" {
				generatedCode := stripe.GenerateSinglePromotionCode(prefix)
				opts.Code = generatedCode
				fmt.Printf("Generated code with prefix '%s': %s\n", prefix, generatedCode)
			}

			// Set optional parameters
			if expiresAt != 0 {
				opts.ExpiresAt = &expiresAt
			}
			if maxRedemptions != 0 {
				opts.MaxRedemptions = &maxRedemptions
			}
			if firstTimeOnly {
				opts.FirstTimeTransaction = &firstTimeOnly
			}
			if minimumAmount != 0 {
				opts.MinimumAmount = &minimumAmount
				opts.Currency = strings.ToLower(currency)
			}
		} else {
			// Use interactive prompts
			opts, err = promptPromoCodeOptions(couponID)
			if err != nil {
				return fmt.Errorf("failed to get promotion code options: %w", err)
			}
		}

		promoService := stripe.NewPromotionCodeService(stripeClient)
		code, err := promoService.CreatePromotionCode(opts)
		if err != nil {
			return fmt.Errorf("failed to create promotion code: %w", err)
		}

		fmt.Printf("✅ Promotion code created successfully!\n")
		fmt.Printf("   ID: %s\n", code.ID)
		fmt.Printf("   Code: %s\n", code.Code)
		fmt.Printf("   Status: %s\n", stripe.FormatPromotionCodeStatus(code))

		return nil
	},
}

var promoBatchCmd = &cobra.Command{
	Use:   "batch <coupon_id>",
	Short: "Batch create promotion codes",
	Long:  "Create multiple promotion codes for an existing coupon.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		couponID := args[0]

		// Get flags
		count, _ := cmd.Flags().GetInt("count")
		prefix, _ := cmd.Flags().GetString("prefix")
		maxRedemptions, _ := cmd.Flags().GetInt64("max-redemptions")

		// Verify coupon exists
		couponService := stripe.NewCouponService(stripeClient)
		coupon, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to verify coupon: %w", err)
		}

		// If no flags provided, prompt for options
		var opts stripe.BatchCreateOptions
		if count == 0 {
			opts, err = promptBatchCreateOptions(couponID)
			if err != nil {
				return fmt.Errorf("failed to get batch options: %w", err)
			}
		} else {
			opts = stripe.BatchCreateOptions{
				CouponID:       couponID,
				Count:          count,
				Prefix:         prefix,
				MaxRedemptions: &maxRedemptions,
			}
		}

		fmt.Printf("Creating %d promotion codes for coupon: %s (%s)\n",
			opts.Count, coupon.ID, stripe.FormatCouponValue(coupon))

		promoService := stripe.NewPromotionCodeService(stripeClient)
		codes, err := promoService.BatchCreatePromotionCodes(opts)

		if err != nil && len(codes) == 0 {
			return fmt.Errorf("failed to create promotion codes: %w", err)
		}

		fmt.Printf("✅ Created %d promotion codes successfully!\n", len(codes))

		if err != nil {
			fmt.Printf("⚠️  Some codes failed to create:\n%v\n", err)
		}

		// Show a few examples
		fmt.Printf("\nExamples:\n")
		for i, code := range codes {
			if i >= 5 { // Show max 5 examples
				fmt.Printf("... and %d more\n", len(codes)-5)
				break
			}
			fmt.Printf("  %s (ID: %s)\n", code.Code, code.ID)
		}

		return nil
	},
}

var promoUpdateCmd = &cobra.Command{
	Use:   "update <promo_id>",
	Short: "Update a promotion code",
	Long: `Update a promotion code's active status and metadata.

Interactive prompts will guide you through:
  • Active status (active or inactive)
  • Metadata updates (planned feature)

Note: Other promotion code properties (code, customer, expiry, etc.) 
cannot be modified after creation per Stripe API limitations.

Examples:
  coupongo promo update promo_1234567890           # Update status interactively
  coupongo promo update promo_1234567890 --env test  # Update in test environment`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promoID := args[0]

		// Get current promotion code
		promoService := stripe.NewPromotionCodeService(stripeClient)
		existing, err := promoService.GetPromotionCode(promoID)
		if err != nil {
			return fmt.Errorf("failed to get existing promotion code: %w", err)
		}

		fmt.Printf("Updating promotion code: %s\n", existing.Code)
		fmt.Printf("Current status: %s\n", stripe.FormatPromotionCodeStatus(existing))

		// Prompt for new status
		statusPrompt := promptui.Select{
			Label: "New status",
			Items: []string{"Active", "Inactive"},
		}

		_, statusChoice, err := statusPrompt.Run()
		if err != nil {
			return err
		}

		active := statusChoice == "Active"

		code, err := promoService.UpdatePromotionCode(promoID, active, nil)
		if err != nil {
			return fmt.Errorf("failed to update promotion code: %w", err)
		}

		fmt.Printf("✅ Promotion code updated successfully!\n")
		fmt.Printf("   Code: %s\n", code.Code)
		fmt.Printf("   Status: %s\n", stripe.FormatPromotionCodeStatus(code))

		return nil
	},
}

func init() {
	// Add subcommands to promo
	promoCmd.AddCommand(promoListCmd)
	promoCmd.AddCommand(promoGetCmd)
	promoCmd.AddCommand(promoCreateCmd)
	promoCmd.AddCommand(promoBatchCmd)
	promoCmd.AddCommand(promoUpdateCmd)

	// Add flags
	promoListCmd.Flags().StringP("coupon", "c", "", "Filter by coupon ID")

	promoBatchCmd.Flags().IntP("count", "n", 0, "Number of promotion codes to create")
	promoBatchCmd.Flags().StringP("prefix", "p", "", "Prefix for promotion codes")
	promoBatchCmd.Flags().Int64("max-redemptions", 0, "Maximum redemptions per code")

	promoCreateCmd.Flags().StringP("prefix", "p", "", "Prefix for promotion code (e.g., BEAR generates BEAR_HUHOIPQW)")
	promoCreateCmd.Flags().StringP("customer", "", "", "Restrict to specific customer ID")
	promoCreateCmd.Flags().BoolP("active", "a", true, "Set promotion code as active (default: true)")
	promoCreateCmd.Flags().Int64P("expires-at", "", 0, "Expiry timestamp (Unix timestamp)")
	promoCreateCmd.Flags().Int64P("max-redemptions", "m", 0, "Maximum redemptions (0 for unlimited)")
	promoCreateCmd.Flags().BoolP("first-time-only", "", false, "Restrict to first-time transactions only")
	promoCreateCmd.Flags().Int64P("minimum-amount", "", 0, "Minimum amount in cents")
	promoCreateCmd.Flags().StringP("currency", "", "usd", "Currency for minimum amount")
}

// promptPromoCodeOptions prompts user for promotion code creation options
func promptPromoCodeOptions(couponID string) (stripe.PromotionCodeCreateOptions, error) {
	var opts stripe.PromotionCodeCreateOptions
	opts.CouponID = couponID

	// Code (optional)
	codePrompt := promptui.Prompt{
		Label: "Promotion code (leave empty for auto-generated)",
	}
	code, _ := codePrompt.Run()
	opts.Code = code

	// Customer (optional)
	customerPrompt := promptui.Prompt{
		Label: "Restrict to specific customer (leave empty to allow any customer, enter customer ID)",
	}
	customer, _ := customerPrompt.Run()
	opts.Customer = customer

	// Active status
	activePrompt := promptui.Select{
		Label: "Set promotion code as active?",
		Items: []string{"Yes", "No"},
	}
	_, activeChoice, err := activePrompt.Run()
	if err != nil {
		return opts, err
	}
	active := activeChoice == "Yes"
	opts.Active = &active

	// Expires at (optional)
	expiresAtPrompt := promptui.Prompt{
		Label: "Expires at timestamp (leave empty for no expiry, format: 1640995200 for 2022-01-01)",
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			val, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timestamp")
			}
			if val <= 0 {
				return fmt.Errorf("timestamp must be greater than 0")
			}
			return nil
		},
	}
	expiresAtStr, _ := expiresAtPrompt.Run()
	if expiresAtStr != "" {
		expiresAt, _ := strconv.ParseInt(expiresAtStr, 10, 64)
		opts.ExpiresAt = &expiresAt
	}

	// Max redemptions (optional)
	maxRedemptionsPrompt := promptui.Prompt{
		Label: "Max redemptions (leave empty for unlimited)",
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			val, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid number")
			}
			if val <= 0 {
				return fmt.Errorf("max redemptions must be greater than 0")
			}
			return nil
		},
	}
	maxRedemptionsStr, _ := maxRedemptionsPrompt.Run()
	if maxRedemptionsStr != "" {
		maxRedemptions, _ := strconv.ParseInt(maxRedemptionsStr, 10, 64)
		opts.MaxRedemptions = &maxRedemptions
	}

	// First-time transaction only
	firstTimePrompt := promptui.Select{
		Label: "Restrict to first-time transactions only?",
		Items: []string{"No", "Yes"},
	}
	_, firstTimeChoice, err := firstTimePrompt.Run()
	if err != nil {
		return opts, err
	}
	if firstTimeChoice == "Yes" {
		firstTime := true
		opts.FirstTimeTransaction = &firstTime
	}

	// Minimum amount (optional)
	minAmountPrompt := promptui.Prompt{
		Label: "Minimum amount in cents (leave empty for no minimum)",
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			val, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid number")
			}
			if val <= 0 {
				return fmt.Errorf("minimum amount must be greater than 0")
			}
			return nil
		},
	}
	minAmountStr, _ := minAmountPrompt.Run()
	if minAmountStr != "" {
		minAmount, _ := strconv.ParseInt(minAmountStr, 10, 64)
		opts.MinimumAmount = &minAmount

		currencyPrompt := promptui.Prompt{
			Label:   "Currency for minimum amount",
			Default: "usd",
		}
		currency, _ := currencyPrompt.Run()
		opts.Currency = strings.ToLower(currency)
	}

	return opts, nil
}

// promptBatchCreateOptions prompts user for batch creation options
func promptBatchCreateOptions(couponID string) (stripe.BatchCreateOptions, error) {
	var opts stripe.BatchCreateOptions
	opts.CouponID = couponID

	// Count
	countPrompt := promptui.Prompt{
		Label: "Number of promotion codes to create (1-1000)",
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("invalid number")
			}
			if val <= 0 || val > 1000 {
				return fmt.Errorf("count must be between 1 and 1000")
			}
			return nil
		},
	}
	countStr, err := countPrompt.Run()
	if err != nil {
		return opts, err
	}
	opts.Count, _ = strconv.Atoi(countStr)

	// Prefix
	prefixPrompt := promptui.Prompt{
		Label:   "Code prefix",
		Default: "PROMO",
	}
	prefix, _ := prefixPrompt.Run()
	opts.Prefix = prefix

	// Max redemptions
	maxRedemptionsPrompt := promptui.Prompt{
		Label: "Max redemptions per code (leave empty for unlimited)",
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			val, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid number")
			}
			if val <= 0 {
				return fmt.Errorf("max redemptions must be greater than 0")
			}
			return nil
		},
	}
	maxRedemptionsStr, _ := maxRedemptionsPrompt.Run()
	if maxRedemptionsStr != "" {
		maxRedemptions, _ := strconv.ParseInt(maxRedemptionsStr, 10, 64)
		opts.MaxRedemptions = &maxRedemptions
	}

	return opts, nil
}
