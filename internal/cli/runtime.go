package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

const schemaVersion = 1

const (
	exitOK        = 0
	exitError     = 1
	exitUsage     = 64
	exitAuth      = 65
	exitNotFound  = 66
	exitConflict  = 67
	exitNetwork   = 68
	exitCancelled = 130
)

var (
	aiFlag      bool
	jsonFlag    bool
	noColorFlag bool
)

type cliError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Code    int    `json:"-"`
}

func (e *cliError) Error() string {
	return e.Message
}

type successEnvelope struct {
	SchemaVersion int         `json:"schema_version"`
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
}

type errorEnvelope struct {
	SchemaVersion int       `json:"schema_version"`
	Success       bool      `json:"success"`
	Error         *cliError `json:"error"`
}

func configureRuntime() {
	color.NoColor = shouldDisableColor()
}

func shouldDisableColor() bool {
	if noColorFlag || aiFlag || jsonFlag {
		return true
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" {
		return true
	}
	if !stdoutIsTerminal() {
		return true
	}
	return false
}

func stdinIsTerminal() bool {
	return isTerminal(os.Stdin)
}

func stdoutIsTerminal() bool {
	return isTerminal(os.Stdout)
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func aiMode() bool {
	return aiFlag || strings.EqualFold(os.Getenv("COUPONGO_AI"), "1")
}

func nonInteractive() bool {
	return aiMode() || os.Getenv("CI") != "" || !stdinIsTerminal()
}

func canPrompt() bool {
	return !nonInteractive()
}

func effectiveOutputFormat(defaultFormat string) OutputFormat {
	if aiMode() || jsonFlag {
		return FormatJSON
	}
	if formatFlag != "" {
		return OutputFormat(formatFlag)
	}
	if !stdoutIsTerminal() || os.Getenv("CI") != "" {
		return FormatJSON
	}
	if defaultFormat != "" {
		return OutputFormat(defaultFormat)
	}
	return FormatTable
}

func effectiveStripeOutputFormat() OutputFormat {
	defaultFormat := ""
	if stripeClient != nil {
		if env, err := stripeClient.GetCurrentEnvironment(); err == nil && env != nil {
			defaultFormat = env.OutputFormat
		}
	}
	return effectiveOutputFormat(defaultFormat)
}

func validateOutputFormat(format string) error {
	if format == "" {
		return nil
	}
	switch OutputFormat(format) {
	case FormatTable, FormatJSON, FormatList:
		return nil
	default:
		return usageError(
			fmt.Sprintf("invalid output format %q", format),
			"use one of: table, json, list",
		)
	}
}

func renderJSON(data interface{}) error {
	if aiMode() {
		return writeJSON(os.Stdout, successEnvelope{
			SchemaVersion: schemaVersion,
			Success:       true,
			Data:          data,
		})
	}
	return writeJSON(os.Stdout, data)
}

func writeJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func usageError(message, hint string) error {
	return &cliError{Kind: "usage", Message: message, Hint: hint, Code: exitUsage}
}

func conflictError(message, hint string) error {
	return &cliError{Kind: "conflict", Message: message, Hint: hint, Code: exitConflict}
}

func notFoundError(message, hint string) error {
	return &cliError{Kind: "not_found", Message: message, Hint: hint, Code: exitNotFound}
}

func cancelledError(message string) error {
	return &cliError{Kind: "cancelled", Message: message, Code: exitCancelled}
}

func exitCodeForError(err error) int {
	var ce *cliError
	if errors.As(err, &ce) && ce.Code != 0 {
		return ce.Code
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "accepts ") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "invalid argument") ||
		strings.Contains(msg, "requires at least") ||
		strings.Contains(msg, "required"):
		return exitUsage
	case strings.Contains(msg, "api key") || strings.Contains(msg, "auth"):
		return exitAuth
	case strings.Contains(msg, "not found") ||
		strings.Contains(msg, "resource_missing") ||
		strings.Contains(msg, "no such "):
		return exitNotFound
	case strings.Contains(msg, "connection") || strings.Contains(msg, "network"):
		return exitNetwork
	default:
		return exitError
	}
}

func normalizeError(err error) *cliError {
	var ce *cliError
	if errors.As(err, &ce) {
		return ce
	}

	kind := "execution"
	code := exitCodeForError(err)
	switch code {
	case exitUsage:
		kind = "usage"
	case exitAuth:
		kind = "auth"
	case exitNotFound:
		kind = "not_found"
	case exitNetwork:
		kind = "network"
	}

	return &cliError{
		Kind:    kind,
		Message: err.Error(),
		Hint:    hintForKind(kind),
		Code:    code,
	}
}

func hintForKind(kind string) string {
	switch kind {
	case "usage":
		return "run `coupongo schema` or the command with `--help` to inspect valid arguments"
	case "auth":
		return "run `coupongo config set-key <environment> --api-key <sk_...>` or choose another environment with `--env`"
	case "not_found":
		return "list the resource first, then retry with a valid ID"
	case "network":
		return "check network connectivity and Stripe API availability, then retry"
	default:
		return ""
	}
}

func renderError(err error) {
	normalized := normalizeError(err)
	if aiMode() {
		_ = writeJSON(os.Stderr, errorEnvelope{
			SchemaVersion: schemaVersion,
			Success:       false,
			Error:         normalized,
		})
		return
	}

	fmt.Fprintf(os.Stderr, "Error: %s\n", normalized.Message)
	if normalized.Hint != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", normalized.Hint)
	}
}

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func parseKeyValueList(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make(map[string]string, len(values))
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, usageError(
				fmt.Sprintf("invalid key-value pair %q", item),
				"use KEY=VALUE, for example `--metadata campaign=spring`",
			)
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result, nil
}

func int64PtrIfPositive(value int64, changed bool, name string) (*int64, error) {
	if !changed {
		return nil, nil
	}
	if value <= 0 {
		return nil, usageError(
			fmt.Sprintf("%s must be greater than 0", name),
			fmt.Sprintf("pass a positive integer for `%s`", name),
		)
	}
	return &value, nil
}
