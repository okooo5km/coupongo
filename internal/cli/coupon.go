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

// couponCmd represents the coupon command
var couponCmd = &cobra.Command{
	Use:   "coupon",
	Short: "Manage Stripe coupons",
	Long:  "Create, list, update, and delete Stripe coupons.",
}

var couponListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all coupons",
	Long:  "List all coupons in the current Stripe account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt64("limit")
		startingAfter, _ := cmd.Flags().GetString("starting-after")
		if limit <= 0 || limit > 100 {
			return usageError("limit must be between 1 and 100", "pass `--limit <1..100>`")
		}

		couponService := stripe.NewCouponService(stripeClient)
		coupons, err := couponService.ListCoupons(limit, startingAfter)
		if err != nil {
			return fmt.Errorf("failed to list coupons: %w", err)
		}

		if len(coupons) == 0 {
			if effectiveStripeOutputFormat() == FormatJSON {
				return renderJSON(coupons)
			}
			fmt.Println("No coupons found.")
			return nil
		}

		renderer := NewOutputRenderer(string(effectiveStripeOutputFormat()))
		return renderer.RenderCoupons(coupons)
	},
}

var couponGetCmd = &cobra.Command{
	Use:   "get <coupon_id>",
	Short: "Get a specific coupon",
	Long:  "Get details of a specific coupon by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		couponID := args[0]
		couponService := stripe.NewCouponService(stripeClient)

		coupon, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to get coupon: %w", err)
		}

		renderer := NewOutputRenderer(string(effectiveStripeOutputFormat()))
		return renderer.RenderCoupon(coupon)
	},
}

var couponCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new coupon",
	Long: `Create a new coupon with specified discount and settings.

Interactive prompts will guide you through:
  • Coupon ID (optional, auto-generated if empty)
  • Coupon name
  • Discount type (percentage or fixed amount)
  • Currency (for fixed amount coupons)
  • Duration (once, forever, or repeating)
  • Duration in months (for repeating coupons)
  • Maximum redemptions (optional)
  • Expiry timestamp (optional)
  • Product restrictions (optional, comma-separated product IDs)
  • Multi-currency amounts (optional, format: eur:950,jpy:1500)

Examples:
  coupongo coupon create                    # Interactive creation
  coupongo coupon create --env production   # Create in production environment
  coupongo coupon create --percent-off 20 --duration once --name "Launch 20"
  coupongo coupon create --amount-off 1500 --currency usd --duration repeating --duration-in-months 3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := couponCreateOptionsFromCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to get coupon options: %w", err)
		}

		couponService := stripe.NewCouponService(stripeClient)
		coupon, err := couponService.CreateCoupon(opts)
		if err != nil {
			return fmt.Errorf("failed to create coupon: %w", err)
		}

		if effectiveStripeOutputFormat() == FormatJSON {
			return renderJSON(coupon)
		}

		fmt.Printf("Coupon created successfully!\n")
		fmt.Printf("   ID: %s\n", coupon.ID)
		fmt.Printf("   Value: %s\n", stripe.FormatCouponValue(coupon))
		fmt.Printf("   Duration: %s\n", stripe.FormatCouponDuration(coupon))

		return nil
	},
}

var couponUpdateCmd = &cobra.Command{
	Use:   "update <coupon_id>",
	Short: "Update a coupon",
	Long: `Update a coupon's name and metadata. Note: discount values cannot be changed after creation.

Interactive prompts will guide you through:
  • Coupon name (optional, leave empty to keep current)
  • Metadata updates (planned feature)

Examples:
  coupongo coupon update coupon-1234567890    # Update coupon interactively
  coupongo coupon update coupon-1234567890 --env test  # Update in test environment
  coupongo coupon update coupon-1234567890 --name "Updated name"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		couponID := args[0]
		hasFlags := false
		cmd.Flags().Visit(func(flag *pflag.Flag) {
			hasFlags = true
		})
		if !hasFlags && !canPrompt() {
			return usageError("coupon update requires at least one update flag in non-interactive mode", "pass `--name` or `--metadata KEY=VALUE`")
		}

		// First, get the existing coupon to show current values
		couponService := stripe.NewCouponService(stripeClient)
		existing, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to get existing coupon: %w", err)
		}

		opts, err := couponUpdateOptionsFromCommand(cmd, existing.Name)
		if err != nil {
			return fmt.Errorf("failed to get update options: %w", err)
		}

		coupon, err := couponService.UpdateCoupon(couponID, opts)
		if err != nil {
			return fmt.Errorf("failed to update coupon: %w", err)
		}

		if effectiveStripeOutputFormat() == FormatJSON {
			return renderJSON(coupon)
		}

		fmt.Printf("Coupon updated successfully!\n")
		fmt.Printf("   ID: %s\n", coupon.ID)
		fmt.Printf("   Name: %s\n", coupon.Name)

		return nil
	},
}

var couponDeleteCmd = &cobra.Command{
	Use:   "delete <coupon_id>",
	Short: "Delete a coupon",
	Long:  "Delete a coupon. This cannot be undone.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := handleHelpArgs(cmd, args); handled {
			return err
		}

		couponID := args[0]

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			if !canPrompt() {
				return usageError("coupon delete requires --yes in non-interactive mode", "retry with `--yes` after confirming the deletion is intended")
			}

			prompt := promptui.Select{
				Label: fmt.Sprintf("Are you sure you want to delete coupon '%s'?", couponID),
				Items: []string{"Yes", "No"},
			}

			_, choice, err := prompt.Run()
			if err != nil || choice == "No" {
				return cancelledError("operation cancelled")
			}
		}

		couponService := stripe.NewCouponService(stripeClient)
		if err := couponService.DeleteCoupon(couponID); err != nil {
			return fmt.Errorf("failed to delete coupon: %w", err)
		}

		result := map[string]interface{}{
			"deleted": true,
			"id":      couponID,
		}
		if effectiveStripeOutputFormat() == FormatJSON {
			return renderJSON(result)
		}

		fmt.Printf("Coupon '%s' deleted successfully!\n", couponID)
		return nil
	},
}

func init() {
	// Add subcommands to coupon
	couponCmd.AddCommand(couponListCmd)
	couponCmd.AddCommand(couponGetCmd)
	couponCmd.AddCommand(couponCreateCmd)
	couponCmd.AddCommand(couponUpdateCmd)
	couponCmd.AddCommand(couponDeleteCmd)

	couponListCmd.Flags().Int64("limit", 100, "Maximum coupons to fetch. Required range: 1..100")
	couponListCmd.Flags().String("starting-after", "", "Cursor ID for Stripe pagination")

	couponCreateCmd.Flags().String("id", "", "Coupon ID. Optional; Stripe auto-generates when omitted")
	couponCreateCmd.Flags().String("name", "", "Coupon name")
	couponCreateCmd.Flags().Float64("percent-off", 0, "Percentage discount. Required unless --amount-off is set. Range: 0 < value <= 100")
	couponCreateCmd.Flags().Int64("amount-off", 0, "Fixed discount amount in the smallest currency unit. Required unless --percent-off is set")
	couponCreateCmd.Flags().String("currency", "usd", "Currency for --amount-off. ISO 4217 lowercase code")
	couponCreateCmd.Flags().String("duration", "once", "Coupon duration. One of: once, forever, repeating")
	couponCreateCmd.Flags().Int64("duration-in-months", 0, "Required when --duration repeating")
	couponCreateCmd.Flags().Int64("max-redemptions", 0, "Maximum redemptions. Omit for unlimited")
	couponCreateCmd.Flags().Int64("redeem-by", 0, "Unix timestamp after which the coupon can no longer be redeemed")
	couponCreateCmd.Flags().String("products", "", "Comma-separated Stripe product IDs to restrict the coupon")
	couponCreateCmd.Flags().String("currency-options", "", "Currency-specific amounts, for example eur:950,jpy:1500")
	couponCreateCmd.Flags().StringArray("metadata", nil, "Metadata key-value pair. Repeat as KEY=VALUE")

	couponUpdateCmd.Flags().String("name", "", "New coupon name")
	couponUpdateCmd.Flags().StringArray("metadata", nil, "Metadata key-value pair. Repeat as KEY=VALUE")

	couponDeleteCmd.Flags().Bool("yes", false, "Confirm deletion without an interactive prompt")
}

func couponCreateOptionsFromCommand(cmd *cobra.Command) (stripe.CouponCreateOptions, error) {
	if len(cmd.Flags().Args()) > 0 {
		return stripe.CouponCreateOptions{}, usageError("coupon create does not accept positional arguments", "use named flags such as `--percent-off` or run interactively in a terminal")
	}

	hasFlags := false
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		hasFlags = true
	})
	if !hasFlags && canPrompt() {
		return promptCouponOptions(false)
	}

	opts := stripe.CouponCreateOptions{}
	id, _ := cmd.Flags().GetString("id")
	name, _ := cmd.Flags().GetString("name")
	percent, _ := cmd.Flags().GetFloat64("percent-off")
	amount, _ := cmd.Flags().GetInt64("amount-off")
	currency, _ := cmd.Flags().GetString("currency")
	duration, _ := cmd.Flags().GetString("duration")
	durationMonths, _ := cmd.Flags().GetInt64("duration-in-months")
	maxRedemptions, _ := cmd.Flags().GetInt64("max-redemptions")
	redeemBy, _ := cmd.Flags().GetInt64("redeem-by")
	products, _ := cmd.Flags().GetString("products")
	currencyOptions, _ := cmd.Flags().GetString("currency-options")
	metadataValues, _ := cmd.Flags().GetStringArray("metadata")

	opts.ID = id
	opts.Name = name
	opts.Currency = strings.ToLower(currency)
	opts.Duration = duration

	if cmd.Flags().Changed("percent-off") {
		if percent <= 0 || percent > 100 {
			return opts, usageError("percent-off must be greater than 0 and at most 100", "pass `--percent-off <number>` in the range 0..100")
		}
		opts.PercentOff = &percent
	}
	if cmd.Flags().Changed("amount-off") {
		amountPtr, err := int64PtrIfPositive(amount, true, "--amount-off")
		if err != nil {
			return opts, err
		}
		opts.AmountOff = amountPtr
	}
	if opts.PercentOff == nil && opts.AmountOff == nil {
		return opts, usageError("coupon create requires --percent-off or --amount-off in non-interactive mode", "pass exactly one discount flag")
	}
	if opts.PercentOff != nil && opts.AmountOff != nil {
		return opts, usageError("coupon create accepts only one discount type", "pass either `--percent-off` or `--amount-off`, not both")
	}
	if duration == "repeating" {
		durationPtr, err := int64PtrIfPositive(durationMonths, true, "--duration-in-months")
		if err != nil {
			return opts, err
		}
		opts.DurationInMonths = durationPtr
	}
	if ptr, err := int64PtrIfPositive(maxRedemptions, cmd.Flags().Changed("max-redemptions"), "--max-redemptions"); err != nil {
		return opts, err
	} else {
		opts.MaxRedemptions = ptr
	}
	if ptr, err := int64PtrIfPositive(redeemBy, cmd.Flags().Changed("redeem-by"), "--redeem-by"); err != nil {
		return opts, err
	} else {
		opts.RedeemBy = ptr
	}
	if productIDs := parseCSV(products); len(productIDs) > 0 {
		opts.AppliesTo = &stripe.CouponAppliesToOptions{Products: productIDs}
	}
	if currencyOptions != "" {
		parsed, err := parseCouponCurrencyOptions(currencyOptions)
		if err != nil {
			return opts, err
		}
		opts.CurrencyOptions = parsed
	}
	metadata, err := parseKeyValueList(metadataValues)
	if err != nil {
		return opts, err
	}
	opts.Metadata = metadata

	return opts, nil
}

func couponUpdateOptionsFromCommand(cmd *cobra.Command, currentName string) (stripe.CouponUpdateOptions, error) {
	hasFlags := false
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		hasFlags = true
	})
	if !hasFlags && canPrompt() {
		fmt.Printf("Updating coupon: %s\n", cmd.Flags().Arg(0))
		fmt.Printf("Current name: %s\n", currentName)
		return promptCouponUpdateOptions()
	}
	if !hasFlags {
		return stripe.CouponUpdateOptions{}, usageError("coupon update requires at least one update flag in non-interactive mode", "pass `--name` or `--metadata KEY=VALUE`")
	}

	name, _ := cmd.Flags().GetString("name")
	metadataValues, _ := cmd.Flags().GetStringArray("metadata")
	metadata, err := parseKeyValueList(metadataValues)
	if err != nil {
		return stripe.CouponUpdateOptions{}, err
	}

	return stripe.CouponUpdateOptions{
		Name:     name,
		Metadata: metadata,
	}, nil
}

func parseCouponCurrencyOptions(value string) (map[string]*stripe.CouponCurrencyOptions, error) {
	result := make(map[string]*stripe.CouponCurrencyOptions)
	for _, pair := range parseCSV(value) {
		currency, amountText, ok := strings.Cut(pair, ":")
		if !ok || currency == "" || amountText == "" {
			return nil, usageError("invalid currency-options value", "use comma-separated currency:amount pairs, for example `eur:950,jpy:1500`")
		}
		amount, err := strconv.ParseInt(amountText, 10, 64)
		if err != nil || amount <= 0 {
			return nil, usageError("currency option amount must be a positive integer", "use the smallest currency unit, for example `eur:950`")
		}
		amountCopy := amount
		result[strings.ToLower(currency)] = &stripe.CouponCurrencyOptions{AmountOff: &amountCopy}
	}
	return result, nil
}

// promptCouponOptions prompts user for coupon creation options
func promptCouponOptions(isUpdate bool) (stripe.CouponCreateOptions, error) {
	var opts stripe.CouponCreateOptions

	if !isUpdate {
		// ID (optional)
		idPrompt := promptui.Prompt{
			Label: "Coupon ID (leave empty for auto-generated)",
		}
		id, _ := idPrompt.Run()
		opts.ID = id
	}

	// Name
	namePrompt := promptui.Prompt{
		Label: "Coupon name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		return opts, err
	}
	opts.Name = name

	// Discount type
	discountPrompt := promptui.Select{
		Label: "Discount type",
		Items: []string{"Percentage", "Fixed amount"},
	}
	_, discountType, err := discountPrompt.Run()
	if err != nil {
		return opts, err
	}

	if discountType == "Percentage" {
		percentPrompt := promptui.Prompt{
			Label: "Percentage off (0-100)",
			Validate: func(input string) error {
				val, err := strconv.ParseFloat(input, 64)
				if err != nil {
					return fmt.Errorf("invalid number")
				}
				if val <= 0 || val > 100 {
					return fmt.Errorf("percentage must be between 0 and 100")
				}
				return nil
			},
		}
		percentStr, err := percentPrompt.Run()
		if err != nil {
			return opts, err
		}
		percent, _ := strconv.ParseFloat(percentStr, 64)
		opts.PercentOff = &percent
	} else {
		amountPrompt := promptui.Prompt{
			Label: "Amount off (in cents, e.g., 1000 for $10.00)",
			Validate: func(input string) error {
				val, err := strconv.ParseInt(input, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid number")
				}
				if val <= 0 {
					return fmt.Errorf("amount must be greater than 0")
				}
				return nil
			},
		}
		amountStr, err := amountPrompt.Run()
		if err != nil {
			return opts, err
		}
		amount, _ := strconv.ParseInt(amountStr, 10, 64)
		opts.AmountOff = &amount

		// Currency
		currencyPrompt := promptui.Prompt{
			Label:   "Currency",
			Default: "usd",
		}
		currency, err := currencyPrompt.Run()
		if err != nil {
			return opts, err
		}
		opts.Currency = strings.ToLower(currency)
	}

	// Duration
	durationPrompt := promptui.Select{
		Label: "Duration",
		Items: []string{"once", "forever", "repeating"},
	}
	_, duration, err := durationPrompt.Run()
	if err != nil {
		return opts, err
	}
	opts.Duration = duration

	if duration == "repeating" {
		monthsPrompt := promptui.Prompt{
			Label: "Duration in months",
			Validate: func(input string) error {
				val, err := strconv.ParseInt(input, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid number")
				}
				if val <= 0 {
					return fmt.Errorf("months must be greater than 0")
				}
				return nil
			},
		}
		monthsStr, err := monthsPrompt.Run()
		if err != nil {
			return opts, err
		}
		months, _ := strconv.ParseInt(monthsStr, 10, 64)
		opts.DurationInMonths = &months
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

	// Redeem by (optional)
	redeemByPrompt := promptui.Prompt{
		Label: "Redeem by timestamp (leave empty for no expiry, format: 1640995200 for 2022-01-01)",
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
	redeemByStr, _ := redeemByPrompt.Run()
	if redeemByStr != "" {
		redeemBy, _ := strconv.ParseInt(redeemByStr, 10, 64)
		opts.RedeemBy = &redeemBy
	}

	// Applies to products (optional)
	appliesToPrompt := promptui.Prompt{
		Label: "Apply to specific products? (leave empty to apply to all, enter comma-separated product IDs)",
	}
	appliesToStr, _ := appliesToPrompt.Run()
	if appliesToStr != "" {
		productIDs := strings.Split(appliesToStr, ",")
		for i, id := range productIDs {
			productIDs[i] = strings.TrimSpace(id)
		}
		if len(productIDs) > 0 && productIDs[0] != "" {
			opts.AppliesTo = &stripe.CouponAppliesToOptions{
				Products: productIDs,
			}
		}
	}

	// Currency options (optional) - only for amount_off coupons
	if opts.AmountOff != nil {
		currencyOptionsPrompt := promptui.Prompt{
			Label: "Add currency-specific amounts? (leave empty to skip, format: eur:950,jpy:1500)",
		}
		currencyOptionsStr, _ := currencyOptionsPrompt.Run()
		if currencyOptionsStr != "" {
			opts.CurrencyOptions = make(map[string]*stripe.CouponCurrencyOptions)
			pairs := strings.Split(currencyOptionsStr, ",")
			for _, pair := range pairs {
				parts := strings.Split(strings.TrimSpace(pair), ":")
				if len(parts) == 2 {
					currency := strings.ToLower(strings.TrimSpace(parts[0]))
					amountStr := strings.TrimSpace(parts[1])
					if amount, err := strconv.ParseInt(amountStr, 10, 64); err == nil {
						opts.CurrencyOptions[currency] = &stripe.CouponCurrencyOptions{
							AmountOff: &amount,
						}
					}
				}
			}
		}
	}

	return opts, nil
}

// promptCouponUpdateOptions prompts user for coupon update options
func promptCouponUpdateOptions() (stripe.CouponUpdateOptions, error) {
	var opts stripe.CouponUpdateOptions

	// Name
	namePrompt := promptui.Prompt{
		Label: "New name (leave empty to keep current)",
	}
	name, _ := namePrompt.Run()
	opts.Name = name

	return opts, nil
}
