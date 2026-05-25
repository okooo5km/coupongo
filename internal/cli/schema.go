package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type schemaDocument struct {
	SchemaVersion int             `json:"schema_version"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Summary       string          `json:"summary"`
	Conventions   schemaContract  `json:"conventions"`
	GlobalFlags   []schemaFlag    `json:"global_flags"`
	Commands      []schemaCommand `json:"commands"`
	Errors        []schemaError   `json:"errors"`
}

type schemaContract struct {
	AIFlag           string   `json:"ai_flag"`
	JSONFlag         string   `json:"json_flag"`
	OutputFormats    []string `json:"output_formats"`
	DataStream       string   `json:"data_stream"`
	DiagnosticStream string   `json:"diagnostic_stream"`
}

type schemaCommand struct {
	Path        string       `json:"path"`
	Use         string       `json:"use"`
	Description string       `json:"description"`
	Mutating    bool         `json:"mutating"`
	Arguments   []schemaArg  `json:"arguments,omitempty"`
	Flags       []schemaFlag `json:"flags,omitempty"`
}

type schemaArg struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

type schemaFlag struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type schemaError struct {
	Kind        string `json:"kind"`
	ExitCode    int    `json:"exit_code"`
	Retryable   bool   `json:"retryable"`
	Description string `json:"description"`
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the machine-readable CLI schema",
	Long:  "Print a concise JSON schema for commands, flags, mutation markers, and error kinds.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return renderJSON(buildSchemaDocument())
	},
}

func buildSchemaDocument() schemaDocument {
	return schemaDocument{
		SchemaVersion: schemaVersion,
		Name:          "coupongo",
		Version:       appVersion,
		Summary:       "Manage Stripe coupons and promotion codes from a human terminal, scripts, or AI agents.",
		Conventions: schemaContract{
			AIFlag:           "--ai",
			JSONFlag:         "--json",
			OutputFormats:    []string{"table", "json", "list"},
			DataStream:       "stdout",
			DiagnosticStream: "stderr",
		},
		GlobalFlags: flagsFromSet(rootCmd.PersistentFlags()),
		Commands:    commandSchemas(rootCmd, nil),
		Errors: []schemaError{
			{Kind: "usage", ExitCode: exitUsage, Retryable: false, Description: "Invalid command, flag, argument, or missing non-interactive confirmation."},
			{Kind: "execution", ExitCode: exitError, Retryable: false, Description: "Stripe or local execution failed after arguments were accepted."},
			{Kind: "auth", ExitCode: exitAuth, Retryable: false, Description: "Stripe API key or authentication failed."},
			{Kind: "not_found", ExitCode: exitNotFound, Retryable: false, Description: "Requested environment or Stripe resource was not found."},
			{Kind: "conflict", ExitCode: exitConflict, Retryable: false, Description: "Requested state conflicts with existing local configuration."},
			{Kind: "network", ExitCode: exitNetwork, Retryable: true, Description: "Network or Stripe API availability issue."},
			{Kind: "cancelled", ExitCode: exitCancelled, Retryable: false, Description: "Interactive operation was cancelled."},
		},
	}
}

func commandSchemas(cmd *cobra.Command, parent []string) []schemaCommand {
	var result []schemaCommand
	children := cmd.Commands()
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})

	for _, child := range children {
		if child.Hidden || child.Name() == "help" {
			continue
		}

		pathParts := append(append([]string{}, parent...), child.Name())
		path := strings.Join(pathParts, " ")
		result = append(result, schemaCommand{
			Path:        path,
			Use:         child.UseLine(),
			Description: child.Short,
			Mutating:    mutatingCommand(path),
			Arguments:   argsFromUse(child.Use),
			Flags:       flagsFromSet(child.NonInheritedFlags()),
		})
		result = append(result, commandSchemas(child, pathParts)...)
	}

	return result
}

func flagsFromSet(flags *pflag.FlagSet) []schemaFlag {
	var result []schemaFlag
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		result = append(result, schemaFlag{
			Name:        "--" + flag.Name,
			Shorthand:   shorthand(flag),
			Type:        flag.Value.Type(),
			Default:     flag.DefValue,
			Description: flag.Usage,
		})
	})
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func shorthand(flag *pflag.Flag) string {
	if flag.Shorthand == "" {
		return ""
	}
	return "-" + flag.Shorthand
}

func argsFromUse(use string) []schemaArg {
	fields := strings.Fields(use)
	var args []schemaArg
	for _, field := range fields[1:] {
		required := strings.HasPrefix(field, "<")
		if !required && !strings.HasPrefix(field, "[") {
			continue
		}
		name := strings.Trim(field, "<>[]")
		name = strings.TrimSuffix(name, "...")
		args = append(args, schemaArg{Name: name, Required: required})
	}
	return args
}

func mutatingCommand(path string) bool {
	switch path {
	case "config init", "config use", "config add-env", "config remove-env", "config set-key", "config reset",
		"coupon create", "coupon update", "coupon delete",
		"promo create", "promo batch", "promo update":
		return true
	default:
		return false
	}
}
