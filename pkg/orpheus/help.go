// help.go: automatic help generation for Orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"
	"sort"
	"strings"

	flashflags "github.com/agilira/flash-flags"
)

// HelpGenerator provides automatic help generation for commands and applications.
type HelpGenerator struct {
	app *App
}

// NewHelpGenerator creates a new help generator for the given app.
func NewHelpGenerator(app *App) *HelpGenerator {
	return &HelpGenerator{app: app}
}

// GenerateCommandHelp generates detailed help for a specific command.
func (h *HelpGenerator) GenerateCommandHelp(cmd *Command) string {
	var sb strings.Builder

	// Header
	usage := cmd.Usage()
	if cmd.HasSubcommands() {
		usage = cmd.name + " <subcommand> [flags]"
	}
	sb.WriteString(fmt.Sprintf("Usage: %s %s\n\n", h.app.name, usage))

	// Description
	if cmd.Description() != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", cmd.Description()))
	}

	// Long description (if available)
	if cmd.longDescription != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", cmd.longDescription))
	}

	// Subcommands (if available)
	if cmd.HasSubcommands() {
		sb.WriteString("Available Subcommands:\n")
		subcommands := cmd.GetSubcommands()

		// Sort subcommand names for consistent output
		var names []string
		for name := range subcommands {
			names = append(names, name)
		}
		for i := 0; i < len(names)-1; i++ {
			for j := i + 1; j < len(names); j++ {
				if names[i] > names[j] {
					names[i], names[j] = names[j], names[i]
				}
			}
		}

		for _, name := range names {
			subcmd := subcommands[name]
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", name, subcmd.Description()))
		}
		sb.WriteString("\n")
	}

	// Examples (if available)
	if len(cmd.examples) > 0 {
		sb.WriteString("Examples:\n")
		for _, example := range cmd.examples {
			sb.WriteString(fmt.Sprintf("  %s\n", example))
		}
		sb.WriteString("\n")
	}

	// Command-specific flags
	if h.hasCommandFlags(cmd) {
		sb.WriteString("Flags:\n")
		sb.WriteString(h.generateFlagHelp(cmd))
		sb.WriteString("\n")
	}

	// Global flags
	sb.WriteString("Global Flags:\n")
	sb.WriteString(h.generateGlobalFlagHelp())

	return sb.String()
}

// GenerateAppHelp generates the main application help.
func (h *HelpGenerator) GenerateAppHelp() string {
	var sb strings.Builder

	// Header with description
	if h.app.description != "" {
		sb.WriteString(h.app.description + "\n\n")
	}

	sb.WriteString(fmt.Sprintf("Usage: %s [command] [flags]\n\n", h.app.name))

	// Available commands (sorted)
	if len(h.app.commands) > 0 {
		sb.WriteString("Available Commands:\n")

		// Sort commands by name
		var names []string
		for name := range h.app.commands {
			names = append(names, name)
		}
		sort.Strings(names)

		// Find longest command name for alignment
		maxLen := 0
		for _, name := range names {
			if len(name) > maxLen {
				maxLen = len(name)
			}
		}

		// Add commands with descriptions
		for _, name := range names {
			cmd := h.app.commands[name]
			padding := strings.Repeat(" ", maxLen-len(name)+2)
			sb.WriteString(fmt.Sprintf("  %s%s%s\n", name, padding, cmd.Description()))
		}

		// Add built-in help command
		padding := strings.Repeat(" ", maxLen-4+2)
		sb.WriteString(fmt.Sprintf("  help%sShow help for commands\n", padding))
		sb.WriteString("\n")
	}

	// Global flags
	sb.WriteString("Global Flags:\n")
	sb.WriteString(h.generateGlobalFlagHelp())
	sb.WriteString("\n")

	// Footer
	sb.WriteString(fmt.Sprintf("Use \"%s help [command]\" for more information about a command.\n", h.app.name))

	return sb.String()
}

// generateGlobalFlagHelp generates help text for global flags.
func (h *HelpGenerator) generateGlobalFlagHelp() string {
	var sb strings.Builder

	// Built-in flags
	sb.WriteString("  -h, --help      Show help\n")
	if h.app.version != "" {
		sb.WriteString("  -v, --version   Show version\n")
	}

	// Custom global flags from flash-flags
	if h.app.globalFlags != nil {
		h.app.globalFlags.VisitAll(func(flag *flashflags.Flag) {
			sb.WriteString(h.formatFlagHelp(flag))
		})
	}

	return sb.String()
}

// generateFlagHelp generates help text for command-specific flags.
func (h *HelpGenerator) generateFlagHelp(cmd *Command) string {
	var sb strings.Builder

	// Command-specific flags from flash-flags
	if cmd.Flags() != nil {
		cmd.Flags().VisitAll(func(flag *flashflags.Flag) {
			sb.WriteString(h.formatFlagHelp(flag))
		})
	}

	// Always show help flag for commands
	sb.WriteString("  -h, --help      Show help for this command\n")

	return sb.String()
}

// hasCommandFlags checks if a command has any flags defined.
func (h *HelpGenerator) hasCommandFlags(cmd *Command) bool {
	if cmd.Flags() == nil {
		return false
	}

	hasFlags := false
	cmd.Flags().VisitAll(func(flag *flashflags.Flag) {
		hasFlags = true
	})
	return hasFlags
}

// formatFlagHelp formats a flash-flags Flag for help output.
func (h *HelpGenerator) formatFlagHelp(flag *flashflags.Flag) string {
	var line strings.Builder

	// Build flag name with short key
	line.WriteString("  ")
	// Note: flash-flags doesn't expose shortKey directly, so we'll show long form
	line.WriteString("--")
	line.WriteString(flag.Name())

	// Add type info for non-bool flags
	if flag.Type() != "bool" {
		line.WriteString(" ")
		line.WriteString(strings.ToUpper(flag.Type()))
	}

	// Pad to align descriptions
	for line.Len() < 30 {
		line.WriteString(" ")
	}

	// Add description
	line.WriteString(flag.Usage())

	// Add default value for non-bool flags
	if flag.Type() != "bool" && flag.Value() != nil {
		line.WriteString(" (default: ")
		line.WriteString(fmt.Sprintf("%v", flag.Value()))
		line.WriteString(")")
	}

	line.WriteString("\n")
	return line.String()
}

// SetLongDescription sets a detailed description for the command.
func (c *Command) SetLongDescription(description string) *Command {
	c.longDescription = description
	return c
}

// AddExample adds a usage example for the command.
func (c *Command) AddExample(example string) *Command {
	c.examples = append(c.examples, example)
	return c
}

// GetHelpGenerator returns the help generator for the application.
func (app *App) GetHelpGenerator() *HelpGenerator {
	return NewHelpGenerator(app)
}
