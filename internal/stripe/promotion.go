package stripe

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/promotioncode"
)

// PromotionCodeService handles promotion code operations
type PromotionCodeService struct {
	client *Client
}

// NewPromotionCodeService creates a new promotion code service
func NewPromotionCodeService(client *Client) *PromotionCodeService {
	return &PromotionCodeService{client: client}
}

// PromotionCodeCreateOptions holds options for creating a promotion code
type PromotionCodeCreateOptions struct {
	CouponID             string
	Code                 string
	Active               *bool
	Customer             string
	MaxRedemptions       *int64
	MinimumAmount        *int64
	Currency             string
	ExpiresAt            *int64
	FirstTimeTransaction *bool
	Metadata             map[string]string
	Restrictions         *PromotionCodeRestrictions
}

// PromotionCodeRestrictions holds restriction options for promotion codes
type PromotionCodeRestrictions struct {
	FirstTimeTransaction *bool
	MinimumAmount        *int64
	Currency             string
}

// BatchCreateOptions holds options for batch creating promotion codes
type BatchCreateOptions struct {
	CouponID             string
	Count                int
	Prefix               string
	Customer             string
	MaxRedemptions       *int64
	MinimumAmount        *int64
	Currency             string
	ExpiresAt            *int64
	FirstTimeTransaction *bool
	Metadata             map[string]string
}

// ListPromotionCodes lists promotion codes, optionally filtered by coupon
func (pcs *PromotionCodeService) ListPromotionCodes(couponID string) ([]*stripe.PromotionCode, error) {
	if !pcs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	params := &stripe.PromotionCodeListParams{}
	params.Filters.AddFilter("limit", "", "100")

	if couponID != "" {
		params.Filters.AddFilter("coupon", "", couponID)
	}

	var codes []*stripe.PromotionCode

	iter := promotioncode.List(params)
	for iter.Next() {
		codes = append(codes, iter.PromotionCode())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list promotion codes: %w", err)
	}

	return codes, nil
}

// GetPromotionCode retrieves a promotion code by ID
func (pcs *PromotionCodeService) GetPromotionCode(id string) (*stripe.PromotionCode, error) {
	if !pcs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	pc, err := promotioncode.Get(id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion code %s: %w", id, err)
	}

	return pc, nil
}

// CreatePromotionCode creates a new promotion code
func (pcs *PromotionCodeService) CreatePromotionCode(opts PromotionCodeCreateOptions) (*stripe.PromotionCode, error) {
	if !pcs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	if opts.CouponID == "" {
		return nil, fmt.Errorf("coupon ID is required")
	}

	params := &stripe.PromotionCodeParams{
		Coupon: stripe.String(opts.CouponID),
	}

	if opts.Code != "" {
		params.Code = stripe.String(opts.Code)
	}

	if opts.Active != nil {
		params.Active = stripe.Bool(*opts.Active)
	}

	if opts.Customer != "" {
		params.Customer = stripe.String(opts.Customer)
	}

	if opts.MaxRedemptions != nil {
		params.MaxRedemptions = stripe.Int64(*opts.MaxRedemptions)
	}

	if opts.ExpiresAt != nil {
		params.ExpiresAt = stripe.Int64(*opts.ExpiresAt)
	}

	if opts.Metadata != nil {
		params.Metadata = opts.Metadata
	}

	// Handle restrictions
	if opts.Restrictions != nil || opts.MinimumAmount != nil || opts.FirstTimeTransaction != nil {
		params.Restrictions = &stripe.PromotionCodeRestrictionsParams{}

		if opts.FirstTimeTransaction != nil {
			params.Restrictions.FirstTimeTransaction = stripe.Bool(*opts.FirstTimeTransaction)
		}

		if opts.MinimumAmount != nil {
			params.Restrictions.MinimumAmount = stripe.Int64(*opts.MinimumAmount)
			if opts.Currency != "" {
				params.Restrictions.MinimumAmountCurrency = stripe.String(opts.Currency)
			}
		}
	}

	pc, err := promotioncode.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create promotion code: %w", err)
	}

	return pc, nil
}

// UpdatePromotionCode updates a promotion code
func (pcs *PromotionCodeService) UpdatePromotionCode(id string, active bool, metadata map[string]string) (*stripe.PromotionCode, error) {
	if !pcs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	params := &stripe.PromotionCodeParams{
		Active: stripe.Bool(active),
	}

	if metadata != nil {
		params.Metadata = metadata
	}

	pc, err := promotioncode.Update(id, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update promotion code %s: %w", id, err)
	}

	return pc, nil
}

// BatchCreatePromotionCodes creates multiple promotion codes for a coupon
func (pcs *PromotionCodeService) BatchCreatePromotionCodes(opts BatchCreateOptions) ([]*stripe.PromotionCode, error) {
	if !pcs.client.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	if opts.CouponID == "" {
		return nil, fmt.Errorf("coupon ID is required")
	}

	if opts.Count <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}

	if opts.Count > 1000 {
		return nil, fmt.Errorf("count cannot exceed 1000")
	}

	var codes []*stripe.PromotionCode
	var errors []error

	for i := 0; i < opts.Count; i++ {
		code := generatePromotionCode(opts.Prefix, i+1)

		createOpts := PromotionCodeCreateOptions{
			CouponID:             opts.CouponID,
			Code:                 code,
			Customer:             opts.Customer,
			MaxRedemptions:       opts.MaxRedemptions,
			MinimumAmount:        opts.MinimumAmount,
			Currency:             opts.Currency,
			ExpiresAt:            opts.ExpiresAt,
			FirstTimeTransaction: opts.FirstTimeTransaction,
			Metadata:             opts.Metadata,
		}

		pc, err := pcs.CreatePromotionCode(createOpts)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create code %s: %w", code, err))
			continue
		}

		codes = append(codes, pc)
	}

	if len(errors) > 0 {
		// Return partial success with errors
		errorMsg := fmt.Sprintf("created %d/%d codes successfully", len(codes), opts.Count)
		for _, err := range errors[:min(len(errors), 5)] { // Show first 5 errors
			errorMsg += fmt.Sprintf("\n  %v", err)
		}
		if len(errors) > 5 {
			errorMsg += fmt.Sprintf("\n  ... and %d more errors", len(errors)-5)
		}
		return codes, fmt.Errorf("%s", errorMsg)
	}

	return codes, nil
}

// generatePromotionCode generates a unique promotion code
func generatePromotionCode(prefix string, index int) string {
	if prefix == "" {
		prefix = "PROMO"
	}

	// Generate a random suffix to make it unique
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(index)))
	suffix := rng.Intn(100000)

	return fmt.Sprintf("%s%d_%05d", strings.ToUpper(prefix), index, suffix)
}

// GenerateSinglePromotionCode generates a single promotion code with 8-char suffix
func GenerateSinglePromotionCode(prefix string) string {
	if prefix == "" {
		prefix = "PROMO"
	}

	// Generate 8 random characters (A-Z, 0-9)
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	suffix := make([]byte, 8)
	for i := range suffix {
		suffix[i] = charset[rng.Intn(len(charset))]
	}

	return fmt.Sprintf("%s_%s", strings.ToUpper(prefix), string(suffix))
}

// FormatPromotionCodeStatus returns a formatted status string
func FormatPromotionCodeStatus(pc *stripe.PromotionCode) string {
	if !pc.Active {
		return "Inactive"
	}

	if pc.ExpiresAt > 0 && pc.ExpiresAt < time.Now().Unix() {
		return "Expired"
	}

	if pc.MaxRedemptions > 0 && pc.TimesRedeemed >= pc.MaxRedemptions {
		return "Max redemptions reached"
	}

	return "Active"
}

// FormatPromotionCodeRedemptions returns a formatted redemption string
func FormatPromotionCodeRedemptions(pc *stripe.PromotionCode) string {
	if pc.MaxRedemptions > 0 {
		return fmt.Sprintf("%d/%d", pc.TimesRedeemed, pc.MaxRedemptions)
	}
	return fmt.Sprintf("%d/unlimited", pc.TimesRedeemed)
}

// FormatPromotionCodeExpiry returns a formatted expiry string
func FormatPromotionCodeExpiry(pc *stripe.PromotionCode) string {
	if pc.ExpiresAt == 0 {
		return "Never"
	}

	t := time.Unix(pc.ExpiresAt, 0)
	return t.Format("2006-01-02 15:04")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
