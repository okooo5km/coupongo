package cli

import (
	"fmt"
	"strconv"
	"strings"

	"coupongo/internal/stripe"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
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
		couponService := stripe.NewCouponService(stripeClient)
		coupons, err := couponService.ListCoupons()
		if err != nil {
			return fmt.Errorf("failed to list coupons: %w", err)
		}

		if len(coupons) == 0 {
			fmt.Println("No coupons found.")
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
		return renderer.RenderCoupons(coupons)
	},
}

var couponGetCmd = &cobra.Command{
	Use:   "get <coupon_id>",
	Short: "Get a specific coupon",
	Long:  "Get details of a specific coupon by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		couponID := args[0]
		couponService := stripe.NewCouponService(stripeClient)

		coupon, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to get coupon: %w", err)
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
  coupongo coupon create --env production   # Create in production environment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := promptCouponOptions(false)
		if err != nil {
			return fmt.Errorf("failed to get coupon options: %w", err)
		}

		couponService := stripe.NewCouponService(stripeClient)
		coupon, err := couponService.CreateCoupon(opts)
		if err != nil {
			return fmt.Errorf("failed to create coupon: %w", err)
		}

		fmt.Printf("✅ Coupon created successfully!\n")
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
  coupongo coupon update coup_1234567890    # Update coupon interactively
  coupongo coupon update coup_1234567890 --env test  # Update in test environment`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		couponID := args[0]

		// First, get the existing coupon to show current values
		couponService := stripe.NewCouponService(stripeClient)
		existing, err := couponService.GetCoupon(couponID)
		if err != nil {
			return fmt.Errorf("failed to get existing coupon: %w", err)
		}

		fmt.Printf("Updating coupon: %s\n", couponID)
		fmt.Printf("Current name: %s\n", existing.Name)

		opts, err := promptCouponUpdateOptions()
		if err != nil {
			return fmt.Errorf("failed to get update options: %w", err)
		}

		coupon, err := couponService.UpdateCoupon(couponID, opts)
		if err != nil {
			return fmt.Errorf("failed to update coupon: %w", err)
		}

		fmt.Printf("✅ Coupon updated successfully!\n")
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
		couponID := args[0]

		// Confirm deletion
		prompt := promptui.Select{
			Label: fmt.Sprintf("Are you sure you want to delete coupon '%s'?", couponID),
			Items: []string{"Yes", "No"},
		}

		_, choice, err := prompt.Run()
		if err != nil || choice == "No" {
			fmt.Println("Operation cancelled.")
			return nil
		}

		couponService := stripe.NewCouponService(stripeClient)
		if err := couponService.DeleteCoupon(couponID); err != nil {
			return fmt.Errorf("failed to delete coupon: %w", err)
		}

		fmt.Printf("✅ Coupon '%s' deleted successfully!\n", couponID)
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
