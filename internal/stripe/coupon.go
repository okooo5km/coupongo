package stripe

import (
	"fmt"
	"strconv"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/coupon"
)

// CouponService handles coupon operations
type CouponService struct {
	client *Client
}

// NewCouponService creates a new coupon service
func NewCouponService(client *Client) *CouponService {
	return &CouponService{client: client}
}

// CouponCreateOptions holds options for creating a coupon
type CouponCreateOptions struct {
	ID               string
	Name             string
	PercentOff       *float64
	AmountOff        *int64
	Currency         string
	Duration         string
	DurationInMonths *int64
	MaxRedemptions   *int64
	RedeemBy         *int64
	AppliesTo        *CouponAppliesToOptions
	CurrencyOptions  map[string]*CouponCurrencyOptions
	Metadata         map[string]string
}

// CouponAppliesToOptions holds applies_to options for a coupon
type CouponAppliesToOptions struct {
	Products []string
}

// CouponCurrencyOptions holds currency-specific options for a coupon
type CouponCurrencyOptions struct {
	AmountOff *int64
}

// CouponUpdateOptions holds options for updating a coupon
type CouponUpdateOptions struct {
	Name     string
	Metadata map[string]string
}

// ListCoupons lists all coupons
func (cs *CouponService) ListCoupons() ([]*stripe.Coupon, error) {
	if !cs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	params := &stripe.CouponListParams{}
	params.Filters.AddFilter("limit", "", "100")

	var coupons []*stripe.Coupon

	iter := coupon.List(params)
	for iter.Next() {
		coupons = append(coupons, iter.Coupon())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list coupons: %w", err)
	}

	return coupons, nil
}

// GetCoupon retrieves a coupon by ID
func (cs *CouponService) GetCoupon(id string) (*stripe.Coupon, error) {
	if !cs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	c, err := coupon.Get(id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get coupon %s: %w", id, err)
	}

	return c, nil
}

// CreateCoupon creates a new coupon
func (cs *CouponService) CreateCoupon(opts CouponCreateOptions) (*stripe.Coupon, error) {
	if !cs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	// Validate options
	if opts.PercentOff == nil && opts.AmountOff == nil {
		return nil, fmt.Errorf("either percent_off or amount_off must be specified")
	}

	if opts.PercentOff != nil && opts.AmountOff != nil {
		return nil, fmt.Errorf("cannot specify both percent_off and amount_off")
	}

	if opts.AmountOff != nil && opts.Currency == "" {
		return nil, fmt.Errorf("currency is required when amount_off is specified")
	}

	if opts.Duration == "" {
		opts.Duration = "once"
	}

	// Validate duration
	validDurations := map[string]bool{
		"forever":   true,
		"once":      true,
		"repeating": true,
	}
	if !validDurations[opts.Duration] {
		return nil, fmt.Errorf("invalid duration: %s (must be forever, once, or repeating)", opts.Duration)
	}

	if opts.Duration == "repeating" && opts.DurationInMonths == nil {
		return nil, fmt.Errorf("duration_in_months is required when duration is repeating")
	}

	params := &stripe.CouponParams{
		Duration: stripe.String(opts.Duration),
	}

	if opts.ID != "" {
		params.ID = stripe.String(opts.ID)
	}

	if opts.Name != "" {
		params.Name = stripe.String(opts.Name)
	}

	if opts.PercentOff != nil {
		params.PercentOff = stripe.Float64(*opts.PercentOff)
	}

	if opts.AmountOff != nil {
		params.AmountOff = stripe.Int64(*opts.AmountOff)
		params.Currency = stripe.String(opts.Currency)
	}

	if opts.DurationInMonths != nil {
		params.DurationInMonths = stripe.Int64(*opts.DurationInMonths)
	}

	if opts.MaxRedemptions != nil {
		params.MaxRedemptions = stripe.Int64(*opts.MaxRedemptions)
	}

	if opts.RedeemBy != nil {
		params.RedeemBy = stripe.Int64(*opts.RedeemBy)
	}

	if opts.Metadata != nil {
		params.Metadata = opts.Metadata
	}

	if opts.AppliesTo != nil && len(opts.AppliesTo.Products) > 0 {
		params.AppliesTo = &stripe.CouponAppliesToParams{
			Products: stripe.StringSlice(opts.AppliesTo.Products),
		}
	}

	if opts.CurrencyOptions != nil {
		params.CurrencyOptions = make(map[string]*stripe.CouponCurrencyOptionsParams)
		for currency, options := range opts.CurrencyOptions {
			if options != nil && options.AmountOff != nil {
				params.CurrencyOptions[currency] = &stripe.CouponCurrencyOptionsParams{
					AmountOff: stripe.Int64(*options.AmountOff),
				}
			}
		}
	}

	c, err := coupon.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create coupon: %w", err)
	}

	return c, nil
}

// UpdateCoupon updates a coupon
func (cs *CouponService) UpdateCoupon(id string, opts CouponUpdateOptions) (*stripe.Coupon, error) {
	if !cs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	params := &stripe.CouponParams{}

	if opts.Name != "" {
		params.Name = stripe.String(opts.Name)
	}

	if opts.Metadata != nil {
		params.Metadata = opts.Metadata
	}

	c, err := coupon.Update(id, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update coupon %s: %w", id, err)
	}

	return c, nil
}

// DeleteCoupon deletes a coupon
func (cs *CouponService) DeleteCoupon(id string) error {
	if !cs.client.IsInitialized() {
		return fmt.Errorf("client not initialized")
	}

	_, err := coupon.Del(id, nil)
	if err != nil {
		return fmt.Errorf("failed to delete coupon %s: %w", id, err)
	}

	return nil
}

// FormatCouponValue returns a formatted string representation of the coupon value
func FormatCouponValue(c *stripe.Coupon) string {
	if c.PercentOff > 0 {
		return fmt.Sprintf("%.1f%% off", c.PercentOff)
	}
	if c.AmountOff > 0 {
		return fmt.Sprintf("%s %s off", formatAmount(c.AmountOff, string(c.Currency)), string(c.Currency))
	}
	return "Unknown discount"
}

// FormatCouponDuration returns a formatted string representation of the coupon duration
func FormatCouponDuration(c *stripe.Coupon) string {
	switch c.Duration {
	case "forever":
		return "Forever"
	case "once":
		return "One time"
	case "repeating":
		return fmt.Sprintf("%d months", c.DurationInMonths)
	default:
		return string(c.Duration)
	}
}

// formatAmount formats an amount in cents to a decimal representation
func formatAmount(amountCents int64, currency string) string {
	// Most currencies use 2 decimal places, but some like JPY use 0
	decimalPlaces := 2
	if currency == "jpy" || currency == "krw" || currency == "vnd" {
		decimalPlaces = 0
	}

	if decimalPlaces == 0 {
		return strconv.FormatInt(amountCents, 10)
	}

	amount := float64(amountCents) / 100.0
	return fmt.Sprintf("%.2f", amount)
}
