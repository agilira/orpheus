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
	defaults          flagDefaults
	parsed            bool
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
	c.defaults.record(name, defaultValue)
	return c
}

// AddBoolFlag adds a boolean flag to the command.
func (c *Command) AddBoolFlag(name, shorthand string, defaultValue bool, description string) *Command {
	if shorthand != "" {
		c.flags.BoolVar(name, shorthand, defaultValue, description)
	} else {
		c.flags.Bool(name, defaultValue, description)
	}
	c.defaults.record(name, defaultValue)
	return c
}

// AddIntFlag adds an integer flag to the command.
func (c *Command) AddIntFlag(name, shorthand string, defaultValue int, description string) *Command {
	if shorthand != "" {
		c.flags.IntVar(name, shorthand, defaultValue, description)
	} else {
		c.flags.Int(name, defaultValue, description)
	}
	c.defaults.record(name, defaultValue)
	return c
}

// AddFloat64Flag adds a float64 flag to the command.
// An empty shorthand registers the long form only.
func (c *Command) AddFloat64Flag(name, shorthand string, defaultValue float64, description string) *Command {
	c.flags.Float64Var(name, shorthand, defaultValue, description)
	c.defaults.record(name, defaultValue)
	return c
}

// AddStringSliceFlag adds a string slice flag to the command.
// An empty shorthand registers the long form only.
func (c *Command) AddStringSliceFlag(name, shorthand string, defaultValue []string, description string) *Command {
	c.flags.StringSliceVar(name, shorthand, defaultValue, description)
	c.defaults.record(name, append([]string(nil), defaultValue...))
	return c
}

// Execute runs the command with the given context.
//
// ctx.Args carries the arguments that follow the command's own name: the
// dispatcher removes it before calling, and so does the descent into a
// subcommand. Execute therefore parses ctx.Args as given.
//
// WHY it no longer strips a leading token matching the command name: by the
// time Execute runs, that name is already gone, so a leading "copy" under the
// command "copy" is a positional argument the user typed. Removing it made
// `myapp copy copy dest` lose an argument -- silently, and only in
// ctx.Positional(), which disagreed with ctx.Args about what had been passed.
//
// WHY the subcommand is matched before help is looked for: the help scan used
// to run first over every argument, so `app remote add -h` printed the help of
// remote, and the help of add could not be reached at all. A subcommand now
// receives its arguments, help flags included, and answers for itself.
func (c *Command) Execute(ctx *Context) error {
	argsToparse := ctx.Args

	// Descend into a named subcommand; it handles its own help
	if subcmd := c.matchSubcommand(argsToparse); subcmd != nil {
		return subcmd.Execute(ctx.forSubcommand(subcmd, argsToparse[1:]))
	}

	// Check for help flags before parsing
	if c.hasHelpFlag(argsToparse) {
		return c.showHelp(ctx)
	}

	// An unknown subcommand name is an error, not a positional argument
	if err := c.checkUnknownSubcommand(argsToparse); err != nil {
		return err
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

// hasHelpFlag checks if help flag is present in args
//
// WHY stop at "--": everything after the POSIX end-of-options marker belongs to
// positional arguments, which flash-flags already honors. A command that forwards
// its arguments to another program ("app run -- tool -h") must hand -h to that
// program instead of printing its own help.
//
// The scan is two-step so the common case stays one pass of string
// comparisons: only when a help spelling appears at all is it checked against
// what the parser will do with it -- see helpRequested.
func (c *Command) hasHelpFlag(args []string) bool {
	// WHY no stop at "--" here: a "--" may be a flag's value, which only
	// helpRequested can tell; it stops at a "--" that really ends the flags.
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return c.helpRequested(args)
		}
	}
	return false
}

// helpRequested reports whether args, parsed as this command's flags, ask for
// the built-in help.
//
// WHY it follows the parser: a help spelling the command registered as its own
// flag (-h for --host) is that flag, and a token consumed as the value of a
// flag (--pattern -h) is that value. Treating either as help printed the help
// text and never ran the handler.
func (c *Command) helpRequested(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return false
		case c.isBuiltinHelp(arg):
			return true
		case c.takesValue(arg):
			i++ // skip the value
		}
	}
	return false
}

// isBuiltinHelp reports whether arg is a help spelling the command left free.
func (c *Command) isBuiltinHelp(arg string) bool {
	switch arg {
	case "--help":
		return !flagNameTaken(c.flags, "help")
	case "-h":
		return !shortKeyTaken(c.flags, "h")
	default:
		return false
	}
}

// takesValue reports whether arg is a flag of this command that consumes the
// next argument as its value: a long flag without "=", or a short flag or
// cluster whose last key is not boolean.
func (c *Command) takesValue(arg string) bool {
	if len(arg) < 2 || arg[0] != '-' || strings.ContainsRune(arg, '=') {
		return false
	}
	var flag *flashflags.Flag
	if strings.HasPrefix(arg, "--") {
		flag = c.flags.Lookup(arg[2:])
	} else {
		flag = shortFlag(c.flags, arg[len(arg)-1:])
	}
	return flag != nil && flag.Type() != "bool"
}

// shortFlag returns the flag of fs registered under shortKey, or nil.
func shortFlag(fs *flashflags.FlagSet, shortKey string) *flashflags.Flag {
	var found *flashflags.Flag
	fs.VisitAll(func(flag *flashflags.Flag) {
		if flag.ShortKey() == shortKey {
			found = flag
		}
	})
	return found
}

// matchSubcommand returns the subcommand named by the first argument, or nil.
// The first non-flag argument is always matched as a subcommand name.
func (c *Command) matchSubcommand(args []string) *Command {
	if !c.HasSubcommands() || len(args) == 0 {
		return nil
	}
	return c.GetSubcommand(args[0])
}

// checkUnknownSubcommand reports a first argument that is not a flag and names
// no subcommand.
// WHY: if no subcommand matches, it is treated as an error -- not as a
// positional argument. Commands that need both subcommands and free positional
// args should use explicit subcommands for all variants (e.g., "stash save",
// "stash pop") to avoid ambiguity.
func (c *Command) checkUnknownSubcommand(args []string) error {
	if !c.HasSubcommands() || len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return nil
	}
	// The error names the parent command; the message names only the
	// subcommand, since Error() already prints the command.
	return NotFoundError(c.name, fmt.Sprintf("unknown subcommand '%s'", args[0]))
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
	if err := parseFresh(c.flags, &c.parsed, args); err != nil {
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

// FullName returns the full command path (e.g. "git remote add").
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

// parseFresh parses args into fs as a new run.
//
// WHY: flash-flags keeps parsed values between calls to Parse, so a second Run
// on the same App saw the first run's flags -- values, FlagChanged and
// GlobalFlagChanged all reported what an earlier invocation had typed.
// FlagSet.Reset restores every default and lets the next Parse load the
// config file again (flash-flags v1.3.4), so each run starts as the first did.
//
// parsed records that fs has been parsed before: the first parse -- the only
// one a CLI process normally makes -- has nothing to restore.
func parseFresh(fs *flashflags.FlagSet, parsed *bool, args []string) error {
	if *parsed {
		fs.Reset()
	}
	*parsed = true
	return fs.Parse(args)
}

// flagDefaults records the default each flag was registered with, by name.
//
// WHY: help used to print a flag's current value as its default, so once a
// value had been parsed -- `app --config x.yaml help` -- the help reported
// "(default: x.yaml)". flash-flags exposes no default after a value is set, so
// Orpheus keeps the one it was given.
type flagDefaults map[string]interface{}

// record stores the default of the named flag, creating the map on first use.
func (d *flagDefaults) record(name string, value interface{}) {
	if *d == nil {
		*d = make(flagDefaults)
	}
	(*d)[name] = value
}

// of returns the recorded default of flag. A flag registered on the FlagSet
// directly, not through Orpheus, has none recorded; its value is the only one
// known.
func (d flagDefaults) of(flag *flashflags.Flag) interface{} {
	if v, ok := d[flag.Name()]; ok {
		return v
	}
	return flag.Value()
}
