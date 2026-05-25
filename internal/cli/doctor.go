package cli

import (
	"fmt"
	"os"
	"runtime"
	"sort"

	"github.com/spf13/cobra"
)

type doctorReport struct {
	SchemaVersion      int           `json:"schema_version"`
	Version            string        `json:"version"`
	GoVersion          string        `json:"go_version"`
	Platform           string        `json:"platform"`
	Arch               string        `json:"arch"`
	ConfigPath         string        `json:"config_path"`
	ConfigExists       bool          `json:"config_exists"`
	CurrentEnvironment string        `json:"current_environment,omitempty"`
	Environments       []doctorEnv   `json:"environments,omitempty"`
	Checks             []doctorCheck `json:"checks"`
}

type doctorEnv struct {
	Name            string `json:"name"`
	Current         bool   `json:"current"`
	APIKey          string `json:"api_key"`
	HasAPIKey       bool   `json:"has_api_key"`
	DefaultCurrency string `json:"default_currency"`
	OutputFormat    string `json:"output_format"`
}

type doctorCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local coupongo readiness",
	Long:  "Check local configuration, environment defaults, and optionally Stripe connectivity.",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkStripe, _ := cmd.Flags().GetBool("check-stripe")
		report := buildDoctorReport(checkStripe)

		if effectiveOutputFormat("") == FormatJSON {
			return renderJSON(report)
		}

		fmt.Printf("CouponGo %s\n", report.Version)
		fmt.Printf("Config: %s\n", report.ConfigPath)
		for _, check := range report.Checks {
			status := "OK"
			if !check.OK {
				status = "FAIL"
			}
			fmt.Printf("%s  %s - %s\n", status, check.Name, check.Message)
			if check.Hint != "" {
				fmt.Printf("    Hint: %s\n", check.Hint)
			}
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().Bool("check-stripe", false, "Make a lightweight Stripe API request using the current environment")
}

func buildDoctorReport(checkStripe bool) doctorReport {
	path := configManager.FilePath()
	report := doctorReport{
		SchemaVersion: schemaVersion,
		Version:       appVersion,
		GoVersion:     runtime.Version(),
		Platform:      runtime.GOOS,
		Arch:          runtime.GOARCH,
		ConfigPath:    path,
		Checks:        []doctorCheck{},
	}

	if _, err := os.Stat(path); err != nil {
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "config",
			OK:      false,
			Message: "configuration file was not found",
			Hint:    "run `coupongo config init` or `coupongo config init --api-key <sk_...> --skip-test`",
		})
		return report
	}

	report.ConfigExists = true
	if err := configManager.Load(); err != nil {
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "config",
			OK:      false,
			Message: err.Error(),
			Hint:    "inspect or remove the config file, then run `coupongo config init`",
		})
		return report
	}

	cfg := configManager.GetConfig()
	report.CurrentEnvironment = cfg.CurrentEnvironment
	report.Checks = append(report.Checks, doctorCheck{
		Name:    "config",
		OK:      true,
		Message: "configuration file is readable",
	})

	envNames := configManager.ListEnvironments()
	sort.Strings(envNames)
	for _, name := range envNames {
		env, err := configManager.GetEnvironment(name)
		if err != nil {
			continue
		}
		report.Environments = append(report.Environments, doctorEnv{
			Name:            name,
			Current:         name == cfg.CurrentEnvironment,
			APIKey:          maskAPIKey(env.StripeAPIKey),
			HasAPIKey:       env.StripeAPIKey != "",
			DefaultCurrency: env.DefaultCurrency,
			OutputFormat:    env.OutputFormat,
		})
	}

	currentEnv, err := configManager.GetCurrentEnvironmentConfig()
	if err != nil {
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "current_environment",
			OK:      false,
			Message: err.Error(),
			Hint:    "run `coupongo config use <environment>` with an existing environment",
		})
		return report
	}
	if currentEnv.StripeAPIKey == "" {
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "api_key",
			OK:      false,
			Message: "current environment has no Stripe API key",
			Hint:    "run `coupongo config set-key " + cfg.CurrentEnvironment + " --api-key <sk_...>`",
		})
	} else {
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "api_key",
			OK:      true,
			Message: "current environment has a Stripe API key",
		})
	}

	if checkStripe {
		if err := stripeClient.Initialize(cfg.CurrentEnvironment); err != nil {
			report.Checks = append(report.Checks, doctorCheck{
				Name:    "stripe",
				OK:      false,
				Message: err.Error(),
				Hint:    "check the current environment API key",
			})
		} else if err := stripeClient.TestConnection(); err != nil {
			report.Checks = append(report.Checks, doctorCheck{
				Name:    "stripe",
				OK:      false,
				Message: err.Error(),
				Hint:    "verify network access and Stripe API key permissions",
			})
		} else {
			report.Checks = append(report.Checks, doctorCheck{
				Name:    "stripe",
				OK:      true,
				Message: "Stripe API request succeeded",
			})
		}
	}

	return report
}
