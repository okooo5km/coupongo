package cli

import (
	"fmt"
	"strconv"
	"strings"

	"coupongo/internal/stripe"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		limit, _ := cmd.Flags().GetInt64("limit")
		startingAfter, _ := cmd.Flags().GetString("starting-after")
		if limit <= 0 || limit > 100 {
			return usageError("limit must be between 1 and 100", "pass `--limit <1..100>`")
		}

		promoService := stripe.NewPromotionCodeService(stripeClient)
		codes, err := promoService.ListPromotionCodes(couponID, limit, startingAfter)
		if err != nil {
			return fmt.Errorf("failed to list promotion codes: %w", err)
		}

		if len(codes) == 0 {
			if effectiveStripeOutputFormat() == FormatJSON {
				return renderJSON(codes)
			}
			if couponID != "" {
				fmt.Printf("No promotion codes found for coupon: %s\n", couponID)
			} else {
				fmt.Println("No promotion codes found.")
			}
			return nil
		}

		renderer := NewOutputRenderer(string(effectiveStripeOutputFormat()))
		return renderer.RenderPromotionCodes(codes)
	},
}

var promoGetCmd = &cobra.Command{
	Use:   "get <promo_id>",
	Short: "Get a specific promotion code",
	Long:  "Get details of a specific promotion code by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		promoID := args[0]
		promoService := stripe.NewPromotionCodeService(stripeClient)

		code, err := promoService.GetPromotionCode(promoID)
		if err != nil {
			return fmt.Errorf("failed to get promotion code: %w", err)
		}

		renderer := NewOutputRenderer(string(effectiveStripeOutputFormat()))
		return renderer.RenderPromotionCode(code)
	},
}

var promoCreateCmd = &cobra.Command{
	Use:   "create <coupon_id>",
	Short: "Create a promotion code",
	Long: `Create a promotion code for an existing coupon.

Use flags for quick creation or no flags for interactive prompts.

Available flags:
  --prefix, -p           Prefix for auto-generated code (e.g., BEAR -> BEAR-XXXXXXXX)
  --code                 Exact promotion code to create
  --separator            Separator between prefix and generated suffix (default '-', use '' for none)
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
  coupongo promo create coupon-1234567890                                    # Interactive creation
  coupongo promo create coupon-1234567890 --code SAVE20                      # Exact code
  coupongo promo create coupon-1234567890 --prefix SAVE                      # Auto-generate with prefix
  coupongo promo create coupon-1234567890 --prefix BEAR --separator ''       # Auto-generate without separator
  coupongo promo create coupon-1234567890 --prefix BEAR --max-redemptions 100  # With limits
  coupongo promo create coupon-1234567890 --customer customer-abc123 --active=false  # Customer-specific, inactive`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		couponID := args[0]

		opts, err := promoCreateOptionsFromCommand(cmd, couponID)
		if err != nil {
			return fmt.Errorf("failed to get promotion code options: %w", err)
		}

		// Verify coupon exists
		couponService := stripe.NewCouponService(stripeClient)
		coupon, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to verify coupon: %w", err)
		}

		if effectiveStripeOutputFormat() != FormatJSON {
			fmt.Printf("Creating promotion code for coupon: %s (%s)\n", coupon.ID, stripe.FormatCouponValue(coupon))
		}

		promoService := stripe.NewPromotionCodeService(stripeClient)
		code, err := promoService.CreatePromotionCode(opts)
		if err != nil {
			return fmt.Errorf("failed to create promotion code: %w", err)
		}

		if effectiveStripeOutputFormat() == FormatJSON {
			return renderJSON(code)
		}

		fmt.Printf("Promotion code created successfully!\n")
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
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		couponID := args[0]

		opts, err := promoBatchOptionsFromCommand(cmd, couponID)
		if err != nil {
			return fmt.Errorf("failed to get batch options: %w", err)
		}

		// Verify coupon exists
		couponService := stripe.NewCouponService(stripeClient)
		coupon, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to verify coupon: %w", err)
		}

		if effectiveStripeOutputFormat() != FormatJSON {
			fmt.Printf("Creating %d promotion codes for coupon: %s (%s)\n",
				opts.Count, coupon.ID, stripe.FormatCouponValue(coupon))
		}

		promoService := stripe.NewPromotionCodeService(stripeClient)
		codes, err := promoService.BatchCreatePromotionCodes(opts)

		if err != nil && len(codes) == 0 {
			return fmt.Errorf("failed to create promotion codes: %w", err)
		}

		result := map[string]interface{}{
			"created": len(codes),
			"codes":   codes,
		}
		if err != nil {
			result["partial_error"] = err.Error()
		}
		if effectiveStripeOutputFormat() == FormatJSON {
			if err != nil && len(codes) > 0 {
				return renderJSON(result)
			}
			return renderJSON(result)
		}

		fmt.Printf("Created %d promotion codes successfully!\n", len(codes))

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
  coupongo promo update promo-1234567890           # Update status interactively
  coupongo promo update promo-1234567890 --env test  # Update in test environment`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		promoID := args[0]

		if !cmd.Flags().Changed("active") && !canPrompt() {
			return usageError("promo update requires --active in non-interactive mode", "pass `--active=true` or `--active=false`")
		}

		// Get current promotion code
		promoService := stripe.NewPromotionCodeService(stripeClient)
		existing, err := promoService.GetPromotionCode(promoID)
		if err != nil {
			return fmt.Errorf("failed to get existing promotion code: %w", err)
		}

		active, err := promoUpdateActiveFromCommand(cmd, existing.Code, stripe.FormatPromotionCodeStatus(existing))
		if err != nil {
			return err
		}

		code, err := promoService.UpdatePromotionCode(promoID, active, nil)
		if err != nil {
			return fmt.Errorf("failed to update promotion code: %w", err)
		}

		if effectiveStripeOutputFormat() == FormatJSON {
			return renderJSON(code)
		}

		fmt.Printf("Promotion code updated successfully!\n")
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
	promoListCmd.Flags().Int64("limit", 100, "Maximum promotion codes to fetch. Required range: 1..100")
	promoListCmd.Flags().String("starting-after", "", "Cursor ID for Stripe pagination")

	promoBatchCmd.Flags().IntP("count", "n", 0, "Number of promotion codes to create")
	promoBatchCmd.Flags().StringP("prefix", "p", "", "Prefix for promotion codes")
	promoBatchCmd.Flags().String("separator", "-", "Separator between prefix and generated content (use '' for none)")
	promoBatchCmd.Flags().Int64("max-redemptions", 0, "Maximum redemptions per code")
	promoBatchCmd.Flags().StringP("customer", "", "", "Restrict each code to a specific customer ID")
	promoBatchCmd.Flags().Int64P("expires-at", "", 0, "Expiry timestamp (Unix timestamp)")
	promoBatchCmd.Flags().BoolP("first-time-only", "", false, "Restrict to first-time transactions only")
	promoBatchCmd.Flags().Int64P("minimum-amount", "", 0, "Minimum amount in cents")
	promoBatchCmd.Flags().StringP("currency", "", "usd", "Currency for minimum amount")
	promoBatchCmd.Flags().StringArray("metadata", nil, "Metadata key-value pair. Repeat as KEY=VALUE")

	promoCreateCmd.Flags().String("code", "", "Exact promotion code to create")
	promoCreateCmd.Flags().StringP("prefix", "p", "", "Prefix for promotion code (e.g., BEAR generates BEAR-HUHOIPQW)")
	promoCreateCmd.Flags().String("separator", "-", "Separator between prefix and generated suffix (use '' for none)")
	promoCreateCmd.Flags().StringP("customer", "", "", "Restrict to specific customer ID")
	promoCreateCmd.Flags().BoolP("active", "a", true, "Set promotion code as active (default: true)")
	promoCreateCmd.Flags().Int64P("expires-at", "", 0, "Expiry timestamp (Unix timestamp)")
	promoCreateCmd.Flags().Int64P("max-redemptions", "m", 0, "Maximum redemptions (0 for unlimited)")
	promoCreateCmd.Flags().BoolP("first-time-only", "", false, "Restrict to first-time transactions only")
	promoCreateCmd.Flags().Int64P("minimum-amount", "", 0, "Minimum amount in cents")
	promoCreateCmd.Flags().StringP("currency", "", "usd", "Currency for minimum amount")
	promoCreateCmd.Flags().StringArray("metadata", nil, "Metadata key-value pair. Repeat as KEY=VALUE")

	promoUpdateCmd.Flags().Bool("active", true, "New active status. Required in non-interactive mode")
}

func promoCreateOptionsFromCommand(cmd *cobra.Command, couponID string) (stripe.PromotionCodeCreateOptions, error) {
	hasFlags := false
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		hasFlags = true
	})
	if !hasFlags && canPrompt() {
		return promptPromoCodeOptions(couponID)
	}

	code, _ := cmd.Flags().GetString("code")
	prefix, _ := cmd.Flags().GetString("prefix")
	separator, _ := cmd.Flags().GetString("separator")
	customer, _ := cmd.Flags().GetString("customer")
	active, _ := cmd.Flags().GetBool("active")
	expiresAt, _ := cmd.Flags().GetInt64("expires-at")
	maxRedemptions, _ := cmd.Flags().GetInt64("max-redemptions")
	firstTimeOnly, _ := cmd.Flags().GetBool("first-time-only")
	minimumAmount, _ := cmd.Flags().GetInt64("minimum-amount")
	currency, _ := cmd.Flags().GetString("currency")
	metadataValues, _ := cmd.Flags().GetStringArray("metadata")

	if separator != "" && separator != "-" {
		return stripe.PromotionCodeCreateOptions{}, usageError("separator must be '-' or empty", "pass `--separator '-'` or `--separator ''`")
	}
	if code != "" && prefix != "" {
		return stripe.PromotionCodeCreateOptions{}, usageError("promo create accepts --code or --prefix, not both", "use `--code` for an exact code or `--prefix` for generated codes")
	}

	opts := stripe.PromotionCodeCreateOptions{
		CouponID: couponID,
		Code:     code,
		Customer: customer,
	}
	if cmd.Flags().Changed("active") || !canPrompt() {
		opts.Active = &active
	}
	if prefix != "" {
		opts.Code = stripe.GenerateSinglePromotionCode(prefix, separator)
	}
	if ptr, err := int64PtrIfPositive(expiresAt, cmd.Flags().Changed("expires-at"), "--expires-at"); err != nil {
		return opts, err
	} else {
		opts.ExpiresAt = ptr
	}
	if ptr, err := int64PtrIfPositive(maxRedemptions, cmd.Flags().Changed("max-redemptions"), "--max-redemptions"); err != nil {
		return opts, err
	} else {
		opts.MaxRedemptions = ptr
	}
	if firstTimeOnly {
		opts.FirstTimeTransaction = &firstTimeOnly
	}
	if ptr, err := int64PtrIfPositive(minimumAmount, cmd.Flags().Changed("minimum-amount"), "--minimum-amount"); err != nil {
		return opts, err
	} else if ptr != nil {
		opts.MinimumAmount = ptr
		opts.Currency = strings.ToLower(currency)
	}
	metadata, err := parseKeyValueList(metadataValues)
	if err != nil {
		return opts, err
	}
	opts.Metadata = metadata

	return opts, nil
}

func promoBatchOptionsFromCommand(cmd *cobra.Command, couponID string) (stripe.BatchCreateOptions, error) {
	count, _ := cmd.Flags().GetInt("count")
	if count == 0 && canPrompt() {
		return promptBatchCreateOptions(couponID)
	}
	if count <= 0 {
		return stripe.BatchCreateOptions{}, usageError("promo batch requires --count in non-interactive mode", "pass `--count <1..1000>`")
	}
	if count > 1000 {
		return stripe.BatchCreateOptions{}, usageError("count cannot exceed 1000", "pass a smaller `--count` value")
	}

	prefix, _ := cmd.Flags().GetString("prefix")
	separator, _ := cmd.Flags().GetString("separator")
	customer, _ := cmd.Flags().GetString("customer")
	maxRedemptions, _ := cmd.Flags().GetInt64("max-redemptions")
	expiresAt, _ := cmd.Flags().GetInt64("expires-at")
	firstTimeOnly, _ := cmd.Flags().GetBool("first-time-only")
	minimumAmount, _ := cmd.Flags().GetInt64("minimum-amount")
	currency, _ := cmd.Flags().GetString("currency")
	metadataValues, _ := cmd.Flags().GetStringArray("metadata")

	if separator != "" && separator != "-" {
		return stripe.BatchCreateOptions{}, usageError("separator must be '-' or empty", "pass `--separator '-'` or `--separator ''`")
	}

	opts := stripe.BatchCreateOptions{
		CouponID:  couponID,
		Count:     count,
		Prefix:    prefix,
		Separator: separator,
		Customer:  customer,
	}
	if ptr, err := int64PtrIfPositive(maxRedemptions, cmd.Flags().Changed("max-redemptions"), "--max-redemptions"); err != nil {
		return opts, err
	} else {
		opts.MaxRedemptions = ptr
	}
	if ptr, err := int64PtrIfPositive(expiresAt, cmd.Flags().Changed("expires-at"), "--expires-at"); err != nil {
		return opts, err
	} else {
		opts.ExpiresAt = ptr
	}
	if firstTimeOnly {
		opts.FirstTimeTransaction = &firstTimeOnly
	}
	if ptr, err := int64PtrIfPositive(minimumAmount, cmd.Flags().Changed("minimum-amount"), "--minimum-amount"); err != nil {
		return opts, err
	} else if ptr != nil {
		opts.MinimumAmount = ptr
		opts.Currency = strings.ToLower(currency)
	}
	metadata, err := parseKeyValueList(metadataValues)
	if err != nil {
		return opts, err
	}
	opts.Metadata = metadata

	return opts, nil
}

func promoUpdateActiveFromCommand(cmd *cobra.Command, code, status string) (bool, error) {
	if cmd.Flags().Changed("active") {
		active, _ := cmd.Flags().GetBool("active")
		return active, nil
	}
	if !canPrompt() {
		return false, usageError("promo update requires --active in non-interactive mode", "pass `--active=true` or `--active=false`")
	}

	fmt.Printf("Updating promotion code: %s\n", code)
	fmt.Printf("Current status: %s\n", status)

	statusPrompt := promptui.Select{
		Label: "New status",
		Items: []string{"Active", "Inactive"},
	}

	_, statusChoice, err := statusPrompt.Run()
	if err != nil {
		return false, err
	}

	return statusChoice == "Active", nil
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

	separatorPrompt := promptui.Select{
		Label: "Insert hyphen between prefix and generated parts?",
		Items: []string{"Yes", "No"},
	}
	_, separatorChoice, err := separatorPrompt.Run()
	if err != nil {
		return opts, err
	}
	if separatorChoice == "Yes" {
		opts.Separator = "-"
	}

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
