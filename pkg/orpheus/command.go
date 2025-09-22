// command.go: commands in Orpheus application framework
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"
	"strings"

	flashflags "github.com/agilira/flash-flags"
)

// CommandHandler is the function signature for command handlers.
type CommandHandler func(ctx *Context) error

// CompletionType represents the type of completion being requested.
type CompletionType int

const (
	// CompletionCommands suggests available commands
	CompletionCommands CompletionType = iota
	// CompletionFlags suggests available flags for a command
	CompletionFlags
	// CompletionArgs suggests possible arguments for a command
	CompletionArgs
)

// CompletionRequest represents a request for tab completion.
type CompletionRequest struct {
	// Type is the type of completion being requested
	Type CompletionType
	// CurrentWord is the word currently being completed
	CurrentWord string
	// Command is the command being completed (if applicable)
	Command string
	// Args are the arguments provided so far
	Args []string
	// Position is the position in the command line
	Position int
}

// CompletionResult represents the result of a completion request.
type CompletionResult struct {
	// Suggestions is the list of completion suggestions
	Suggestions []string
	// Directive provides hints to the shell about how to handle completions
	Directive CompletionDirective
}

// CompletionDirective provides instructions to the shell.
type CompletionDirective int

const (
	// CompletionDefault indicates normal completion behavior
	CompletionDefault CompletionDirective = iota
	// CompletionNoSpace indicates no space should be added after completion
	CompletionNoSpace
	// CompletionNoFiles indicates file completion should be disabled
	CompletionNoFiles
)

// CompletionHandler is a function that provides custom completion for a command.
type CompletionHandler func(req *CompletionRequest) *CompletionResult

// Command represents a CLI command with its configuration and behavior.
type Command struct {
	name              string
	description       string
	longDescription   string
	usage             string
	examples          []string
	flags             *flashflags.FlagSet
	handler           CommandHandler
	completionHandler CompletionHandler
	subcommands       map[string]*Command
	parent            *Command
}

// NewCommand creates a new command with the specified name and description.
func NewCommand(name, description string) *Command {
	return &Command{
		name:        name,
		description: description,
		flags:       flashflags.New(name),
		subcommands: make(map[string]*Command),
	}
}

// Name returns the command name.
func (c *Command) Name() string {
	return c.name
}

// Description returns the command description.
func (c *Command) Description() string {
	return c.description
}

// Usage returns the command usage string.
func (c *Command) Usage() string {
	if c.usage != "" {
		return c.usage
	}
	return c.name + " [flags]"
}

// SetUsage sets a custom usage string for the command.
func (c *Command) SetUsage(usage string) *Command {
	c.usage = usage
	return c
}

// SetHandler sets the command handler function.
func (c *Command) SetHandler(handler CommandHandler) *Command {
	c.handler = handler
	return c
}

// SetCompletionHandler sets the completion handler function.
func (c *Command) SetCompletionHandler(handler CompletionHandler) *Command {
	c.completionHandler = handler
	return c
}

// AddFlag adds a string flag to the command.
func (c *Command) AddFlag(name, shorthand, defaultValue, description string) *Command {
	if shorthand != "" {
		c.flags.StringVar(name, shorthand, defaultValue, description)
	} else {
		c.flags.String(name, defaultValue, description)
	}
	return c
}

// AddBoolFlag adds a boolean flag to the command.
func (c *Command) AddBoolFlag(name, shorthand string, defaultValue bool, description string) *Command {
	if shorthand != "" {
		c.flags.BoolVar(name, shorthand, defaultValue, description)
	} else {
		c.flags.Bool(name, defaultValue, description)
	}
	return c
}

// AddIntFlag adds an integer flag to the command.
func (c *Command) AddIntFlag(name, shorthand string, defaultValue int, description string) *Command {
	if shorthand != "" {
		c.flags.IntVar(name, shorthand, defaultValue, description)
	} else {
		c.flags.Int(name, defaultValue, description)
	}
	return c
}

// AddFloat64Flag adds a float64 flag to the command.
func (c *Command) AddFloat64Flag(name, shorthand string, defaultValue float64, description string) *Command {
	if shorthand != "" {
		// flash-flags doesn't have Float64Var with shorthand, use basic method
		c.flags.Float64(name, defaultValue, description)
	} else {
		c.flags.Float64(name, defaultValue, description)
	}
	return c
}

// AddStringSliceFlag adds a string slice flag to the command.
func (c *Command) AddStringSliceFlag(name, shorthand string, defaultValue []string, description string) *Command {
	if shorthand != "" {
		// flash-flags doesn't have StringSliceVar with shorthand, use basic method
		c.flags.StringSlice(name, defaultValue, description)
	} else {
		c.flags.StringSlice(name, defaultValue, description)
	}
	return c
}

// Execute runs the command with the given context.
func (c *Command) Execute(ctx *Context) error {
	argsToparse := c.prepareArgs(ctx.Args)

	// Check for help flags before parsing
	if c.hasHelpFlag(argsToparse) {
		return c.showHelp(ctx)
	}

	// Handle subcommands if they exist
	subcommandExecuted, err := c.handleSubcommands(ctx, argsToparse)
	if err != nil {
		return err
	}

	// If a subcommand was executed successfully, we're done
	if subcommandExecuted {
		return nil
	}

	// If no subcommand provided and this command has subcommands but no handler, show help
	if c.HasSubcommands() && c.handler == nil {
		return c.showHelp(ctx)
	}

	// Validate handler existence
	if err := c.validateHandler(); err != nil {
		return err
	}

	// Parse and execute
	return c.parseAndExecute(ctx, argsToparse)
}

// prepareArgs removes the command name from args if present
func (c *Command) prepareArgs(args []string) []string {
	if len(args) > 0 && args[0] == c.name {
		return args[1:]
	}
	return args
}

// hasHelpFlag checks if help flag is present in args
func (c *Command) hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

// handleSubcommands processes subcommand execution
func (c *Command) handleSubcommands(ctx *Context, args []string) (bool, error) {
	if !c.HasSubcommands() || len(args) == 0 {
		return false, nil
	}

	potentialSubcmd := args[0]

	// Don't treat flags as subcommands
	if strings.HasPrefix(potentialSubcmd, "-") {
		return false, nil
	}

	if subcmd := c.GetSubcommand(potentialSubcmd); subcmd != nil {
		// Execute subcommand with remaining args
		newCtx := &Context{
			App:         ctx.App,
			Args:        args[1:], // Remove subcommand name
			GlobalFlags: ctx.GlobalFlags,
			Command:     subcmd,
		}
		err := subcmd.Execute(newCtx)
		return true, err // Subcommand was executed
	}

	// Subcommand not found - this is an error
	return false, NotFoundError(c.name+" "+potentialSubcmd, fmt.Sprintf("unknown subcommand '%s' for command '%s'", potentialSubcmd, c.name))
}

// validateHandler checks if the command has a valid handler
func (c *Command) validateHandler() error {
	// If no handler is defined and no subcommands, error
	if c.handler == nil && !c.HasSubcommands() {
		return ExecutionError(c.name, "no handler defined for command")
	}

	return nil
}

// parseAndExecute handles flag parsing and handler execution
func (c *Command) parseAndExecute(ctx *Context, args []string) error {
	// Parse flags for this command
	if err := c.flags.Parse(args); err != nil {
		return ValidationError(c.name, "flag parsing failed: "+err.Error())
	}

	// Update context with parsed flags
	ctx.Flags = c.flags
	ctx.Command = c

	// Execute the handler
	return c.handler(ctx)
}

// Flags returns the command's flag set for advanced usage.
func (c *Command) Flags() *flashflags.FlagSet {
	return c.flags
}

// AddSubcommand adds a subcommand to this command.
func (c *Command) AddSubcommand(cmd *Command) *Command {
	cmd.parent = c
	c.subcommands[cmd.name] = cmd
	return c
}

// Subcommand creates and adds a subcommand with the specified name, description, and handler.
func (c *Command) Subcommand(name, description string, handler CommandHandler) *Command {
	subcmd := NewCommand(name, description).SetHandler(handler)
	c.AddSubcommand(subcmd)
	return subcmd
}

// GetSubcommands returns a copy of the subcommands map for introspection.
func (c *Command) GetSubcommands() map[string]*Command {
	subcommands := make(map[string]*Command)
	for name, cmd := range c.subcommands {
		subcommands[name] = cmd
	}
	return subcommands
}

// HasSubcommands returns true if this command has subcommands.
func (c *Command) HasSubcommands() bool {
	return len(c.subcommands) > 0
}

// GetSubcommand returns a subcommand by name, or nil if not found.
func (c *Command) GetSubcommand(name string) *Command {
	return c.subcommands[name]
}

// Parent returns the parent command, or nil if this is a root command.
func (c *Command) Parent() *Command {
	return c.parent
}

// FullName returns the full command path (e.g., "git remote add").
func (c *Command) FullName() string {
	if c.parent == nil {
		return c.name
	}
	return c.parent.FullName() + " " + c.name
}

// showHelp displays help for the command.
func (c *Command) showHelp(ctx *Context) error {
	generator := NewHelpGenerator(ctx.App)
	helpText := generator.GenerateCommandHelp(c)
	fmt.Print(helpText)
	return nil
}
