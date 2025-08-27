package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"coupongo/internal/stripe"

	"github.com/olekukonko/tablewriter"
	stripe_api "github.com/stripe/stripe-go/v82"
)

// RenderPromotionCodes renders promotion codes in the specified format
func (r *OutputRenderer) RenderPromotionCodes(codes []*stripe_api.PromotionCode) error {
	switch r.format {
	case FormatJSON:
		return r.RenderJSON(codes)
	case FormatList:
		return r.renderPromoCodeList(codes)
	case FormatTable:
		fallthrough
	default:
		return r.renderPromoCodeTable(codes)
	}
}

// RenderPromotionCode renders a single promotion code in the specified format
func (r *OutputRenderer) RenderPromotionCode(code *stripe_api.PromotionCode) error {
	switch r.format {
	case FormatJSON:
		return r.RenderJSON(code)
	case FormatList:
		return r.renderPromoCodeDetails(code)
	case FormatTable:
		fallthrough
	default:
		return r.renderPromoCodeDetails(code)
	}
}

// renderPromoCodeTable renders promotion codes in a beautiful table format
func (r *OutputRenderer) renderPromoCodeTable(codes []*stripe_api.PromotionCode) error {
	table := tablewriter.NewWriter(os.Stdout)

	// Clean table styling
	table.SetHeader([]string{"Code", "Coupon", "Status", "Redeemed", "Expires"})
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
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
	)

	for _, code := range codes {
		// Format status with color
		status := stripe.FormatPromotionCodeStatus(code)
		var coloredStatus string
		switch {
		case !code.Active:
			coloredStatus = red("✗ " + status)
		case code.ExpiresAt > 0 && code.ExpiresAt < time.Now().Unix():
			coloredStatus = yellow("⚠ " + status)
		case code.MaxRedemptions > 0 && code.TimesRedeemed >= code.MaxRedemptions:
			coloredStatus = yellow("⚠ " + status)
		default:
			coloredStatus = green("✓ " + status)
		}

		// Format coupon info
		couponInfo := cyan(code.Coupon.ID)
		if code.Coupon.Name != "" {
			couponInfo = fmt.Sprintf("%s\n(%s)", cyan(code.Coupon.ID), code.Coupon.Name)
		}

		// Format redemption count
		redeemed := stripe.FormatPromotionCodeRedemptions(code)
		if code.MaxRedemptions > 0 && code.TimesRedeemed >= code.MaxRedemptions {
			redeemed = red(redeemed)
		}

		// Format expiry
		expires := stripe.FormatPromotionCodeExpiry(code)
		if expires != "Never" && code.ExpiresAt > 0 && code.ExpiresAt < time.Now().Unix() {
			expires = red(expires)
		} else if expires != "Never" {
			expires = yellow(expires)
		}

		table.Append([]string{
			white(code.Code),
			couponInfo,
			coloredStatus,
			redeemed,
			expires,
		})
	}

	fmt.Printf("\n%s\n", white("🎟️ PROMOTION CODES"))
	table.Render()
	fmt.Printf("\n%s %s\n\n", cyan("Total:"), white(fmt.Sprintf("%d promotion code(s)", len(codes))))

	return nil
}

// renderPromoCodeList renders promotion codes in a beautiful list format
func (r *OutputRenderer) renderPromoCodeList(codes []*stripe_api.PromotionCode) error {
	if len(codes) == 0 {
		fmt.Printf("%s No promotion codes found.\n", yellow("ℹ"))
		return nil
	}

	fmt.Printf("\n%s\n", white("🎟️ PROMOTION CODES"))
	fmt.Println(strings.Repeat("═", 50))

	for i, code := range codes {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 50))
		}

		// Header with code and status
		status := stripe.FormatPromotionCodeStatus(code)
		var statusIcon string
		var statusColor func(...interface{}) string

		switch {
		case !code.Active:
			statusIcon = "✗"
			statusColor = red
		case code.ExpiresAt > 0 && code.ExpiresAt < time.Now().Unix():
			statusIcon = "⚠"
			statusColor = yellow
		case code.MaxRedemptions > 0 && code.TimesRedeemed >= code.MaxRedemptions:
			statusIcon = "⚠"
			statusColor = yellow
		default:
			statusIcon = "✓"
			statusColor = green
		}

		fmt.Printf("%s %s %s %s\n",
			magenta("🎟️"),
			white(code.Code),
			statusIcon,
			statusColor(strings.ToUpper(status)))

		// Coupon info
		fmt.Printf("   %s %s", cyan("Coupon:"), blue(code.Coupon.ID))
		if code.Coupon.Name != "" {
			fmt.Printf(" (%s)", code.Coupon.Name)
		}
		fmt.Println()

		// Discount value
		fmt.Printf("   %s %s\n", cyan("Discount:"), green(stripe.FormatCouponValue(code.Coupon)))

		// Usage stats
		redeemed := stripe.FormatPromotionCodeRedemptions(code)
		if code.MaxRedemptions > 0 && code.TimesRedeemed >= code.MaxRedemptions {
			fmt.Printf("   %s %s %s\n", cyan("Usage:"), red(redeemed), red("(Limit reached)"))
		} else {
			fmt.Printf("   %s %s\n", cyan("Usage:"), redeemed)
		}

		// Created date
		fmt.Printf("   %s %s\n", cyan("Created:"),
			time.Unix(code.Created, 0).Format("2006-01-02 15:04"))

		// Expiry
		if code.ExpiresAt > 0 {
			expiryTime := time.Unix(code.ExpiresAt, 0)
			if expiryTime.Before(time.Now()) {
				fmt.Printf("   %s %s\n", cyan("Expired:"), red(expiryTime.Format("2006-01-02 15:04")))
			} else {
				fmt.Printf("   %s %s\n", cyan("Expires:"), yellow(expiryTime.Format("2006-01-02 15:04")))
			}
		}

		// Restrictions
		if code.Restrictions != nil {
			if code.Restrictions.FirstTimeTransaction {
				fmt.Printf("   %s %s\n", cyan("Restriction:"), yellow("First-time customers only"))
			}
			if code.Restrictions.MinimumAmount > 0 {
				fmt.Printf("   %s %s\n", cyan("Min. Amount:"),
					yellow(fmt.Sprintf("%s %s", formatAmount(code.Restrictions.MinimumAmount, string(code.Restrictions.MinimumAmountCurrency)), strings.ToUpper(string(code.Restrictions.MinimumAmountCurrency)))))
			}
		}
	}

	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("%s %s\n\n", cyan("Total:"), white(fmt.Sprintf("%d promotion code(s)", len(codes))))

	return nil
}

// renderPromoCodeDetails renders detailed information about a single promotion code
func (r *OutputRenderer) renderPromoCodeDetails(code *stripe_api.PromotionCode) error {
	fmt.Printf("\n%s\n", white("🎟️ PROMOTION CODE DETAILS"))
	fmt.Println(strings.Repeat("═", 60))

	// Code and Status
	status := stripe.FormatPromotionCodeStatus(code)
	var statusIcon string
	var statusColor func(...interface{}) string

	switch {
	case !code.Active:
		statusIcon = "✗"
		statusColor = red
	case code.ExpiresAt > 0 && code.ExpiresAt < time.Now().Unix():
		statusIcon = "⚠"
		statusColor = yellow
	case code.MaxRedemptions > 0 && code.TimesRedeemed >= code.MaxRedemptions:
		statusIcon = "⚠"
		statusColor = yellow
	default:
		statusIcon = "✓"
		statusColor = green
	}

	fmt.Printf("%s %s\n", white("Code:"), magenta(code.Code))
	fmt.Printf("%s %s %s\n", white("Status:"), statusIcon, statusColor(strings.ToUpper(status)))
	fmt.Printf("%s %s\n", white("ID:"), gray(code.ID))

	// Coupon information
	fmt.Println()
	fmt.Printf("%s\n", white("🎫 COUPON"))
	fmt.Printf("  %s %s\n", cyan("ID:"), blue(code.Coupon.ID))
	if code.Coupon.Name != "" {
		fmt.Printf("  %s %s\n", cyan("Name:"), code.Coupon.Name)
	}
	fmt.Printf("  %s %s\n", cyan("Discount:"), green(stripe.FormatCouponValue(code.Coupon)))
	fmt.Printf("  %s %s\n", cyan("Duration:"), cyan(stripe.FormatCouponDuration(code.Coupon)))

	// Usage statistics
	fmt.Println()
	fmt.Printf("%s\n", white("📊 USAGE"))
	fmt.Printf("  %s %d\n", cyan("Times Redeemed:"), code.TimesRedeemed)
	if code.MaxRedemptions > 0 {
		fmt.Printf("  %s %d\n", cyan("Max Redemptions:"), code.MaxRedemptions)
		remaining := code.MaxRedemptions - code.TimesRedeemed
		if remaining > 0 {
			fmt.Printf("  %s %s\n", cyan("Remaining:"), green(fmt.Sprintf("%d", remaining)))
		} else {
			fmt.Printf("  %s %s\n", cyan("Remaining:"), red("0 (Limit reached)"))
		}
	} else {
		fmt.Printf("  %s %s\n", cyan("Max Redemptions:"), "Unlimited")
	}

	// Restrictions
	if code.Restrictions != nil {
		fmt.Println()
		fmt.Printf("%s\n", white("🚫 RESTRICTIONS"))
		if code.Restrictions.FirstTimeTransaction {
			fmt.Printf("  %s %s\n", cyan("Customer Type:"), yellow("First-time customers only"))
		}
		if code.Restrictions.MinimumAmount > 0 {
			fmt.Printf("  %s %s %s\n", cyan("Minimum Amount:"),
				yellow(formatAmount(code.Restrictions.MinimumAmount, string(code.Restrictions.MinimumAmountCurrency))),
				cyan(strings.ToUpper(string(code.Restrictions.MinimumAmountCurrency))))
		}
	}

	// Timestamps
	fmt.Println()
	fmt.Printf("%s\n", white("📅 DATES"))
	fmt.Printf("  %s %s\n", cyan("Created:"),
		time.Unix(code.Created, 0).Format("2006-01-02 15:04:05 MST"))

	if code.ExpiresAt > 0 {
		expiryTime := time.Unix(code.ExpiresAt, 0)
		if expiryTime.Before(time.Now()) {
			fmt.Printf("  %s %s\n", cyan("Expired:"), red(expiryTime.Format("2006-01-02 15:04:05 MST")))
		} else {
			fmt.Printf("  %s %s\n", cyan("Expires:"), yellow(expiryTime.Format("2006-01-02 15:04:05 MST")))
		}
	} else {
		fmt.Printf("  %s %s\n", cyan("Expires:"), green("Never"))
	}

	// Metadata
	if len(code.Metadata) > 0 {
		fmt.Println()
		fmt.Printf("%s\n", white("🏷️  METADATA"))
		for key, value := range code.Metadata {
			fmt.Printf("  %s %s\n", cyan(key+":"), value)
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("═", 60))
	return nil
}
