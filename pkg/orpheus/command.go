// command.go: commands in orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"

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
}

// NewCommand creates a new command with the specified name and description.
func NewCommand(name, description string) *Command {
	return &Command{
		name:        name,
		description: description,
		flags:       flashflags.New(name),
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
	if c.handler == nil {
		return NewOrpheusError(ErrorExecution, c.name, "no handler defined for command", 1)
	}

	// Parse command-specific flags from remaining args
	// Skip the command name if it's still in the args
	argsToparse := ctx.Args
	if len(argsToparse) > 0 && argsToparse[0] == c.name {
		argsToparse = argsToparse[1:]
	}

	// Check for help flags before parsing
	for _, arg := range argsToparse {
		if arg == "--help" || arg == "-h" {
			return c.showHelp(ctx)
		}
	}

	if err := c.flags.Parse(argsToparse); err != nil {
		return NewOrpheusError(ErrorValidation, c.name, "flag parsing failed: "+err.Error(), 1)
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

// showHelp displays help for the command.
func (c *Command) showHelp(ctx *Context) error {
	generator := NewHelpGenerator(ctx.App)
	helpText := generator.GenerateCommandHelp(c)
	fmt.Print(helpText)
	return nil
}
