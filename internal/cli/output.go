package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/olekukonko/tablewriter"
	stripe_api "github.com/stripe/stripe-go/v82"
)

// OutputFormat defines the available output formats
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatList  OutputFormat = "list"
)

// OutputRenderer handles different output formats
type OutputRenderer struct {
	format OutputFormat
}

// NewOutputRenderer creates a new output renderer
func NewOutputRenderer(format string) *OutputRenderer {
	if format == "" {
		format = "table"
	}
	return &OutputRenderer{format: OutputFormat(format)}
}

// RenderJSON renders data as pretty-printed JSON with syntax highlighting
func (r *OutputRenderer) RenderJSON(data interface{}) error {
	if r.format != FormatJSON {
		return fmt.Errorf("renderer not in JSON mode")
	}

	// Convert to JSON with proper formatting
	formatter := prettyjson.NewFormatter()
	formatter.Indent = 2
	formatter.KeyColor = color.New(color.FgBlue, color.Bold)
	formatter.StringColor = color.New(color.FgGreen)
	formatter.BoolColor = color.New(color.FgYellow)
	formatter.NumberColor = color.New(color.FgCyan)
	formatter.NullColor = color.New(color.FgBlack, color.Bold)

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	coloredBytes, err := formatter.Format(jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	fmt.Println(string(coloredBytes))
	return nil
}

// Color helper functions
var (
	cyan    = color.New(color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	white   = color.New(color.FgWhite, color.Bold).SprintFunc()
	gray    = color.New(color.FgBlack, color.Bold).SprintFunc()
)

// RenderCoupons renders coupons in the specified format
func (r *OutputRenderer) RenderCoupons(coupons []*stripe_api.Coupon) error {
	switch r.format {
	case FormatJSON:
		return r.RenderJSON(coupons)
	case FormatList:
		return r.renderCouponList(coupons)
	case FormatTable:
		fallthrough
	default:
		return r.renderCouponTable(coupons)
	}
}

// RenderCoupon renders a single coupon in the specified format
func (r *OutputRenderer) RenderCoupon(coupon *stripe_api.Coupon) error {
	switch r.format {
	case FormatJSON:
		return r.RenderJSON(coupon)
	case FormatList:
		return r.renderCouponDetails(coupon)
	case FormatTable:
		fallthrough
	default:
		return r.renderCouponDetails(coupon)
	}
}

// renderCouponTable renders coupons in a beautiful table format
func (r *OutputRenderer) renderCouponTable(coupons []*stripe_api.Coupon) error {
	table := tablewriter.NewWriter(os.Stdout)

	// Clean table styling
	table.SetHeader([]string{"ID", "Name", "Discount", "Duration", "Redeemed", "Status"})
	table.SetBorder(true)
	table.SetHeaderLine(true)
	table.SetRowLine(false)
	table.SetCenterSeparator("+")
	table.SetColumnSeparator("|")
	table.SetRowSeparator("-")
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetColWidth(80)

	// Header colors
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
	)

	for _, coupon := range coupons {
		name := coupon.Name
		if name == "" {
			name = gray("(no name)")
		}

		// Format discount value with color
		var discount string
		if coupon.PercentOff > 0 {
			discount = green(fmt.Sprintf("%.0f%% off", coupon.PercentOff))
		} else if coupon.AmountOff > 0 {
			discount = blue(fmt.Sprintf("%s %s off", formatAmount(coupon.AmountOff, string(coupon.Currency)), strings.ToUpper(string(coupon.Currency))))
		} else {
			discount = gray("Unknown")
		}

		// Format duration with color
		var duration string
		switch coupon.Duration {
		case "forever":
			duration = green("Forever")
		case "once":
			duration = yellow("One time")
		case "repeating":
			duration = cyan(fmt.Sprintf("%d months", coupon.DurationInMonths))
		default:
			duration = string(coupon.Duration)
		}

		// Format redeemed count
		var redeemed string
		if coupon.MaxRedemptions > 0 {
			redeemed = fmt.Sprintf("%d/%d", coupon.TimesRedeemed, coupon.MaxRedemptions)
			if coupon.TimesRedeemed >= coupon.MaxRedemptions {
				redeemed = red(redeemed)
			}
		} else {
			redeemed = fmt.Sprintf("%d/unlimited", coupon.TimesRedeemed)
		}

		// Status with color
		status := green("✓ Active")
		if !coupon.Valid {
			status = red("✗ Invalid")
		}

		// Note: SetRowColor is not available in all versions of tablewriter
		// Colors are already applied to individual cells above

		table.Append([]string{
			cyan(coupon.ID),
			name,
			discount,
			duration,
			redeemed,
			status,
		})
	}

	fmt.Printf("\n%s\n", white("📋 COUPONS"))
	table.Render()
	fmt.Printf("\n%s %s\n\n", cyan("Total:"), white(fmt.Sprintf("%d coupon(s)", len(coupons))))

	return nil
}

// renderCouponList renders coupons in a beautiful list format
func (r *OutputRenderer) renderCouponList(coupons []*stripe_api.Coupon) error {
	if len(coupons) == 0 {
		fmt.Printf("%s No coupons found.\n", yellow("ℹ"))
		return nil
	}

	fmt.Printf("\n%s\n", white("📋 COUPONS"))
	fmt.Println(strings.Repeat("═", 50))

	for i, coupon := range coupons {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 50))
		}

		// Header with ID and status
		status := green("✓ ACTIVE")
		if !coupon.Valid {
			status = red("✗ INVALID")
		}

		fmt.Printf("%s %s %s\n",
			cyan("🎫"),
			white(coupon.ID),
			status)

		// Name
		if coupon.Name != "" {
			fmt.Printf("   %s %s\n", cyan("Name:"), coupon.Name)
		}

		// Discount
		if coupon.PercentOff > 0 {
			fmt.Printf("   %s %s\n", cyan("Discount:"), green(fmt.Sprintf("%.0f%% off", coupon.PercentOff)))
		} else if coupon.AmountOff > 0 {
			fmt.Printf("   %s %s\n", cyan("Discount:"),
				blue(fmt.Sprintf("%s %s off", formatAmount(coupon.AmountOff, string(coupon.Currency)), strings.ToUpper(string(coupon.Currency)))))
		}

		// Duration
		var durationText string
		switch coupon.Duration {
		case "forever":
			durationText = green("Forever")
		case "once":
			durationText = yellow("One time use")
		case "repeating":
			durationText = cyan(fmt.Sprintf("Valid for %d months", coupon.DurationInMonths))
		}
		fmt.Printf("   %s %s\n", cyan("Duration:"), durationText)

		// Usage stats
		if coupon.MaxRedemptions > 0 {
			fmt.Printf("   %s %d/%d", cyan("Usage:"), coupon.TimesRedeemed, coupon.MaxRedemptions)
			if coupon.TimesRedeemed >= coupon.MaxRedemptions {
				fmt.Printf(" %s", red("(Limit reached)"))
			}
			fmt.Println()
		} else {
			fmt.Printf("   %s %d (unlimited)\n", cyan("Usage:"), coupon.TimesRedeemed)
		}

		// Created date
		fmt.Printf("   %s %s\n", cyan("Created:"),
			time.Unix(coupon.Created, 0).Format("2006-01-02 15:04"))

		// Expiry if applicable
		if coupon.RedeemBy > 0 {
			expiryTime := time.Unix(coupon.RedeemBy, 0)
			if expiryTime.Before(time.Now()) {
				fmt.Printf("   %s %s\n", cyan("Expired:"), red(expiryTime.Format("2006-01-02 15:04")))
			} else {
				fmt.Printf("   %s %s\n", cyan("Expires:"), yellow(expiryTime.Format("2006-01-02 15:04")))
			}
		}
	}

	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("%s %s\n\n", cyan("Total:"), white(fmt.Sprintf("%d coupon(s)", len(coupons))))

	return nil
}

// renderCouponDetails renders detailed information about a single coupon
func (r *OutputRenderer) renderCouponDetails(coupon *stripe_api.Coupon) error {
	fmt.Printf("\n%s\n", white("🎫 COUPON DETAILS"))
	fmt.Println(strings.Repeat("═", 60))

	// ID and Status
	status := green("✓ ACTIVE")
	if !coupon.Valid {
		status = red("✗ INVALID")
	}
	fmt.Printf("%s %s\n", white("ID:"), cyan(coupon.ID))
	fmt.Printf("%s %s\n", white("Status:"), status)

	// Name
	if coupon.Name != "" {
		fmt.Printf("%s %s\n", white("Name:"), coupon.Name)
	}

	// Discount details
	fmt.Println()
	fmt.Printf("%s\n", white("💰 DISCOUNT"))
	if coupon.PercentOff > 0 {
		fmt.Printf("  %s %s\n", cyan("Type:"), "Percentage")
		fmt.Printf("  %s %s\n", cyan("Value:"), green(fmt.Sprintf("%.1f%% off", coupon.PercentOff)))
	} else if coupon.AmountOff > 0 {
		fmt.Printf("  %s %s\n", cyan("Type:"), "Fixed Amount")
		fmt.Printf("  %s %s\n", cyan("Value:"),
			blue(fmt.Sprintf("%s %s off", formatAmount(coupon.AmountOff, string(coupon.Currency)), strings.ToUpper(string(coupon.Currency)))))
		fmt.Printf("  %s %s\n", cyan("Currency:"), strings.ToUpper(string(coupon.Currency)))
	}

	// Duration details
	fmt.Println()
	fmt.Printf("%s\n", white("⏰ DURATION"))
	switch coupon.Duration {
	case "forever":
		fmt.Printf("  %s %s\n", cyan("Type:"), green("Forever"))
	case "once":
		fmt.Printf("  %s %s\n", cyan("Type:"), yellow("One time use"))
	case "repeating":
		fmt.Printf("  %s %s\n", cyan("Type:"), "Repeating")
		fmt.Printf("  %s %s\n", cyan("Duration:"), cyan(fmt.Sprintf("%d months", coupon.DurationInMonths)))
	}

	// Usage statistics
	fmt.Println()
	fmt.Printf("%s\n", white("📊 USAGE"))
	fmt.Printf("  %s %d\n", cyan("Times Redeemed:"), coupon.TimesRedeemed)
	if coupon.MaxRedemptions > 0 {
		fmt.Printf("  %s %d\n", cyan("Max Redemptions:"), coupon.MaxRedemptions)
		remaining := coupon.MaxRedemptions - coupon.TimesRedeemed
		if remaining > 0 {
			fmt.Printf("  %s %s\n", cyan("Remaining:"), green(fmt.Sprintf("%d", remaining)))
		} else {
			fmt.Printf("  %s %s\n", cyan("Remaining:"), red("0 (Limit reached)"))
		}
	} else {
		fmt.Printf("  %s %s\n", cyan("Max Redemptions:"), "Unlimited")
	}

	// Timestamps
	fmt.Println()
	fmt.Printf("%s\n", white("📅 DATES"))
	fmt.Printf("  %s %s\n", cyan("Created:"),
		time.Unix(coupon.Created, 0).Format("2006-01-02 15:04:05 MST"))

	if coupon.RedeemBy > 0 {
		expiryTime := time.Unix(coupon.RedeemBy, 0)
		if expiryTime.Before(time.Now()) {
			fmt.Printf("  %s %s\n", cyan("Expired:"), red(expiryTime.Format("2006-01-02 15:04:05 MST")))
		} else {
			fmt.Printf("  %s %s\n", cyan("Expires:"), yellow(expiryTime.Format("2006-01-02 15:04:05 MST")))
		}
	}

	// Metadata
	if len(coupon.Metadata) > 0 {
		fmt.Println()
		fmt.Printf("%s\n", white("🏷️  METADATA"))
		for key, value := range coupon.Metadata {
			fmt.Printf("  %s %s\n", cyan(key+":"), value)
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("═", 60))
	return nil
}

// formatAmount formats an amount in cents to a decimal representation
func formatAmount(amountCents int64, currency string) string {
	// Most currencies use 2 decimal places, but some like JPY use 0
	decimalPlaces := 2
	if currency == "jpy" || currency == "krw" || currency == "vnd" {
		decimalPlaces = 0
	}

	if decimalPlaces == 0 {
		return fmt.Sprintf("%d", amountCents)
	}

	amount := float64(amountCents) / 100.0
	return fmt.Sprintf("%.2f", amount)
}
