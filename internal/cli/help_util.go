package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// handleHelpArgs checks if the provided args request help and renders it.
// Returns true when help was displayed so callers can stop further processing.
func handleHelpArgs(cmd *cobra.Command, args []string) (bool, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true, cmd.Help()
		}
	}

	if len(args) == 1 && strings.EqualFold(args[0], "help") {
		return true, cmd.Help()
	}

	return false, nil
}
