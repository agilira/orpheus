// help.go: automatic help generation in Orpheus application framework
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

	// Build help sections
	h.addCommandUsage(&sb, cmd)
	h.addCommandDescription(&sb, cmd)
	h.addSubcommands(&sb, cmd)
	h.addExamples(&sb, cmd)
	h.addCommandFlags(&sb, cmd)
	h.addGlobalFlags(&sb)

	return sb.String()
}

// addCommandUsage adds the usage line to the help text
func (h *HelpGenerator) addCommandUsage(sb *strings.Builder, cmd *Command) {
	usage := cmd.Usage()
	if cmd.HasSubcommands() {
		usage = cmd.name + " <subcommand> [flags]"
	}
	sb.WriteString(fmt.Sprintf("Usage: %s %s\n\n", h.app.name, usage))
}

// addCommandDescription adds the command description to the help text
func (h *HelpGenerator) addCommandDescription(sb *strings.Builder, cmd *Command) {
	if cmd.Description() != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", cmd.Description()))
	}

	// Long description (if available)
	if cmd.longDescription != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", cmd.longDescription))
	}
}

// addSubcommands adds the subcommands section to the help text
func (h *HelpGenerator) addSubcommands(sb *strings.Builder, cmd *Command) {
	if !cmd.HasSubcommands() {
		return
	}

	sb.WriteString("Available Subcommands:\n")
	subcommands := cmd.GetSubcommands()
	names := h.sortSubcommandNames(subcommands)

	for _, name := range names {
		subcmd := subcommands[name]
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", name, subcmd.Description()))
	}
	sb.WriteString("\n")
}

// sortSubcommandNames sorts subcommand names for consistent output
func (h *HelpGenerator) sortSubcommandNames(subcommands map[string]*Command) []string {
	var names []string
	for name := range subcommands {
		names = append(names, name)
	}

	// Use efficient standard library sorting
	sort.Strings(names)
	return names
}

// addExamples adds the examples section to the help text
func (h *HelpGenerator) addExamples(sb *strings.Builder, cmd *Command) {
	if len(cmd.examples) == 0 {
		return
	}

	sb.WriteString("Examples:\n")
	for _, example := range cmd.examples {
		sb.WriteString(fmt.Sprintf("  %s\n", example))
	}
	sb.WriteString("\n")
}

// addCommandFlags adds the command-specific flags section
func (h *HelpGenerator) addCommandFlags(sb *strings.Builder, cmd *Command) {
	if !h.hasCommandFlags(cmd) {
		return
	}

	sb.WriteString("Flags:\n")
	sb.WriteString(h.generateFlagHelp(cmd))
	sb.WriteString("\n")
}

// addGlobalFlags adds the global flags section
func (h *HelpGenerator) addGlobalFlags(sb *strings.Builder) {
	sb.WriteString("Global Flags:\n")
	sb.WriteString(h.generateGlobalFlagHelp())
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

	// Built-in flags, in whichever spellings the application has left free
	sb.WriteString(builtinFlagHelp(h.app.globalFlags, "help", "h", "Show help"))
	if h.app.version != "" {
		sb.WriteString(builtinFlagHelp(h.app.globalFlags, "version", "v", "Show version"))
	}

	// Custom global flags from flash-flags
	if h.app.globalFlags != nil {
		h.app.globalFlags.VisitAll(func(flag *flashflags.Flag) {
			sb.WriteString(h.formatFlagHelp(flag))
		})
	}

	return sb.String()
}

// builtinFlagHelp renders the help line for a built-in flag, showing only the
// spellings that still reach it.
//
// WHY it asks: an application may register its own --version or -h, and the
// parser then yields those spellings to it. Listing them here anyway would
// describe an interface the program does not have -- and, where the short key
// is claimed, print it twice with two different meanings. A built-in whose
// every spelling is claimed is not listed at all.
func builtinFlagHelp(fs *flashflags.FlagSet, longName, shortKey, description string) string {
	longFree := !flagNameTaken(fs, longName)
	shortFree := !shortKeyTaken(fs, shortKey)

	var spelling string
	switch {
	case longFree && shortFree:
		spelling = fmt.Sprintf("-%s, --%s", shortKey, longName)
	case longFree:
		spelling = "--" + longName
	case shortFree:
		spelling = "-" + shortKey
	default:
		return ""
	}

	return padFlagHelp(spelling) + description + "\n"
}

// padFlagHelp indents a flag spelling and pads it to the description column
// shared by every help line.
func padFlagHelp(spelling string) string {
	var line strings.Builder
	line.WriteString("  ")
	line.WriteString(spelling)
	for line.Len() < flagHelpDescriptionColumn {
		line.WriteString(" ")
	}
	return line.String()
}

// flagHelpDescriptionColumn is the column where every flag description starts.
const flagHelpDescriptionColumn = 30

// generateFlagHelp generates help text for command-specific flags.
func (h *HelpGenerator) generateFlagHelp(cmd *Command) string {
	var sb strings.Builder

	// Command-specific flags from flash-flags
	if cmd.Flags() != nil {
		cmd.Flags().VisitAll(func(flag *flashflags.Flag) {
			sb.WriteString(h.formatFlagHelp(flag))
		})
	}

	// The built-in help flag, in whichever spellings the command has left free
	sb.WriteString(builtinFlagHelp(cmd.Flags(), "help", "h", "Show help for this command"))

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

	// Build flag name with short key.
	//
	// WHY the short key is shown: it was omitted here on the belief that
	// flash-flags did not expose it, so a flag declared with a shorthand
	// worked on the command line and was never mentioned in help -- the one
	// place a user looks to discover it. Flag.ShortKey() reports it.
	line.WriteString("  ")
	if short := flag.ShortKey(); short != "" {
		line.WriteString("-")
		line.WriteString(short)
		line.WriteString(", ")
	}
	line.WriteString("--")
	line.WriteString(flag.Name())

	// Add type info for non-bool flags
	if flag.Type() != "bool" {
		line.WriteString(" ")
		line.WriteString(strings.ToUpper(flag.Type()))
	}

	// Pad to align descriptions
	for line.Len() < flagHelpDescriptionColumn {
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
