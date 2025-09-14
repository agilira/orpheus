// Package orpheus provides an ultra-fast, lightweight CLI framework for Go applications.
//
// Orpheus is designed to be simple, performant, and feature-complete, offering:
//   - Ultra-fast command parsing and execution
//   - Comprehensive flag support via flash-flags integration
//   - Automatic help generation and shell completion
//   - Type-safe command handlers with context
//   - Fluent API for easy application building
//
// Basic usage:
//
//	app := orpheus.New("myapp").
//		SetDescription("My awesome CLI application").
//		SetVersion("1.0.0")
//
//	app.Command("hello", "Say hello", func(ctx *orpheus.Context) error {
//		name := ctx.GetFlagString("name")
//		if name == "" {
//			name = "World"
//		}
//		fmt.Printf("Hello, %s!\n", name)
//		return nil
//	}).AddFlag("name", "n", "", "Name to greet")
//
//	return app.Run(os.Args[1:])
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

// App represents the main CLI application.
type App struct {
	name             string
	description      string
	version          string
	commands         map[string]*Command
	globalFlags      *flashflags.FlagSet
	defaultCmd       string
	helpCommand      *Command
	logger           Logger
	auditLogger      AuditLogger
	tracer           Tracer
	metricsCollector MetricsCollector
}

// New creates a new Orpheus application.
func New(name string) *App {
	app := &App{
		name:        name,
		commands:    make(map[string]*Command),
		globalFlags: flashflags.New(name),
	}

	// Add built-in help command
	app.helpCommand = NewCommand("help", "Show help for commands").
		SetHandler(app.helpHandler)

	return app
}

// SetDescription sets the application description.
func (app *App) SetDescription(description string) *App {
	app.description = description
	return app
}

// SetVersion sets the application version.
func (app *App) SetVersion(version string) *App {
	app.version = version
	return app
}

// SetLogger sets the logger for the application.
func (app *App) SetLogger(logger Logger) *App {
	app.logger = logger
	return app
}

// SetAuditLogger sets the audit logger for the application.
func (app *App) SetAuditLogger(auditLogger AuditLogger) *App {
	app.auditLogger = auditLogger
	return app
}

// SetTracer sets the tracer for the application.
func (app *App) SetTracer(tracer Tracer) *App {
	app.tracer = tracer
	return app
}

// SetMetricsCollector sets the metrics collector for the application.
func (app *App) SetMetricsCollector(collector MetricsCollector) *App {
	app.metricsCollector = collector
	return app
}

// Logger returns the configured logger.
func (app *App) Logger() Logger {
	return app.logger
}

// AuditLogger returns the configured audit logger.
func (app *App) AuditLogger() AuditLogger {
	return app.auditLogger
}

// Tracer returns the configured tracer.
func (app *App) Tracer() Tracer {
	return app.tracer
}

// MetricsCollector returns the configured metrics collector.
func (app *App) MetricsCollector() MetricsCollector {
	return app.metricsCollector
}

// AddGlobalFlag adds a global string flag.
func (app *App) AddGlobalFlag(name, shorthand, defaultValue, description string) *App {
	if shorthand != "" {
		app.globalFlags.StringVar(name, shorthand, defaultValue, description)
	} else {
		app.globalFlags.String(name, defaultValue, description)
	}
	return app
}

// AddGlobalBoolFlag adds a global boolean flag.
func (app *App) AddGlobalBoolFlag(name, shorthand string, defaultValue bool, description string) *App {
	if shorthand != "" {
		app.globalFlags.BoolVar(name, shorthand, defaultValue, description)
	} else {
		app.globalFlags.Bool(name, defaultValue, description)
	}
	return app
}

// AddGlobalIntFlag adds a global integer flag.
func (app *App) AddGlobalIntFlag(name, shorthand string, defaultValue int, description string) *App {
	if shorthand != "" {
		app.globalFlags.IntVar(name, shorthand, defaultValue, description)
	} else {
		app.globalFlags.Int(name, defaultValue, description)
	}
	return app
}

// Command adds a command using a simple handler function.
func (app *App) Command(name, description string, handler CommandHandler) *App {
	cmd := NewCommand(name, description).SetHandler(handler)
	app.commands[name] = cmd
	return app
}

// AddCommand adds a pre-configured command.
func (app *App) AddCommand(cmd *Command) *App {
	app.commands[cmd.Name()] = cmd
	return app
}

// SetDefaultCommand sets the command to run when no command is specified.
func (app *App) SetDefaultCommand(cmdName string) *App {
	app.defaultCmd = cmdName
	return app
}

// Run executes the application with the given arguments.
func (app *App) Run(args []string) error {
	// Handle empty args
	if len(args) == 0 {
		return app.handleEmptyArgs()
	}

	// Handle built-in flags
	if handled, err := app.handleBuiltinFlags(args); handled {
		return err
	}

	// Parse global flags and get command
	globalArgs, cmdArgs := app.splitGlobalArgs(args)
	if err := app.globalFlags.Parse(globalArgs); err != nil {
		return ValidationError("", "global flag parsing failed: "+err.Error())
	}

	// Handle command execution
	return app.handleCommandExecution(cmdArgs)
}

// handleEmptyArgs handles the case when no arguments are provided.
func (app *App) handleEmptyArgs() error {
	if app.defaultCmd != "" {
		return app.runCommand(app.defaultCmd, []string{})
	}
	return app.helpHandler(&Context{App: app})
}

// handleBuiltinFlags handles built-in flags like --help and --version.
func (app *App) handleBuiltinFlags(args []string) (handled bool, err error) {
	firstArg := args[0]

	// Check for global help flag
	if firstArg == "--help" || firstArg == "-h" {
		return true, app.helpHandler(&Context{App: app})
	}

	// Check for version flag
	if firstArg == "--version" || firstArg == "-v" {
		app.printVersion()
		return true, nil
	}

	return false, nil
}

// printVersion prints the application version.
func (app *App) printVersion() {
	if app.version != "" {
		fmt.Printf("%s version %s\n", app.name, app.version)
	} else {
		fmt.Printf("%s (no version set)\n", app.name)
	}
}

// handleCommandExecution handles the execution of commands.
func (app *App) handleCommandExecution(cmdArgs []string) error {
	// Get command name
	if len(cmdArgs) == 0 {
		return app.handleEmptyArgs()
	}

	cmdName := cmdArgs[0]
	cmdArgs = cmdArgs[1:] // Remove command name from args

	// Handle built-in help command
	if cmdName == "help" {
		return app.handleHelpCommand(cmdArgs)
	}

	return app.runCommand(cmdName, cmdArgs)
}

// handleHelpCommand handles the built-in help command.
func (app *App) handleHelpCommand(cmdArgs []string) error {
	if len(cmdArgs) > 0 {
		return app.showCommandHelp(cmdArgs[0])
	}
	return app.helpHandler(&Context{App: app})
}

// runCommand executes a specific command.
func (app *App) runCommand(cmdName string, args []string) error {
	cmd, exists := app.commands[cmdName]
	if !exists {
		return NotFoundError(cmdName, fmt.Sprintf("command '%s' not found", cmdName))
	}

	// Create execution context
	ctx := &Context{
		App:         app,
		Args:        args,
		GlobalFlags: app.globalFlags,
	}

	// Execute the command
	return cmd.Execute(ctx)
}

// splitGlobalArgs separates global flags from command and command args.
func (app *App) splitGlobalArgs(args []string) (globalArgs, cmdArgs []string) {
	var i int
	for i = 0; i < len(args); i++ {
		arg := args[i]

		// Stop at first non-flag argument (the command)
		if !strings.HasPrefix(arg, "-") {
			break
		}

		// Process this flag
		processed, skipNext := app.processSingleFlag(args, i)
		globalArgs = append(globalArgs, processed...)
		if skipNext {
			i++ // Skip the value
		}
	}

	// Everything from the command onwards is command args
	cmdArgs = args[i:]
	return globalArgs, cmdArgs
}

// processSingleFlag processes a single flag and returns the processed args and whether to skip next arg.
func (app *App) processSingleFlag(args []string, i int) (processed []string, skipNext bool) {
	arg := args[i]

	// Check if this is a boolean global flag
	isBoolFlag := app.isBooleanGlobalFlag(arg)

	if isBoolFlag || strings.Contains(arg, "=") {
		// Boolean flag or flag with embedded value
		return []string{arg}, false
	}

	// Flag that might need a value
	if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
		return []string{arg, args[i+1]}, true
	}

	return []string{arg}, false
}

// isBooleanGlobalFlag checks if the given argument is a boolean global flag.
func (app *App) isBooleanGlobalFlag(arg string) bool {
	if strings.HasPrefix(arg, "--") {
		return app.isLongBooleanFlag(arg)
	}

	if len(arg) == 2 && arg[0] == '-' {
		return app.isShortBooleanFlag(arg)
	}

	return false
}

// isLongBooleanFlag checks if a long flag (--flag) is boolean.
func (app *App) isLongBooleanFlag(arg string) bool {
	flagName := arg[2:]
	if eqPos := strings.IndexByte(flagName, '='); eqPos != -1 {
		flagName = flagName[:eqPos]
	}

	if flag := app.globalFlags.Lookup(flagName); flag != nil && flag.Type() == "bool" {
		return true
	}

	return false
}

// isShortBooleanFlag checks if a short flag (-f) is boolean.
func (app *App) isShortBooleanFlag(arg string) bool {
	shortKey := string(arg[1])
	// Treat common boolean short flags as boolean
	return shortKey == "v" || shortKey == "h" || shortKey == "d"
}

// helpHandler handles the help command.
func (app *App) helpHandler(ctx *Context) error {
	generator := NewHelpGenerator(app)
	fmt.Printf("%s", generator.GenerateAppHelp())
	return nil
}

// showCommandHelp shows help for a specific command.
func (app *App) showCommandHelp(cmdName string) error {
	cmd, exists := app.commands[cmdName]
	if !exists {
		return NotFoundError(cmdName, fmt.Sprintf("command '%s' not found", cmdName))
	}

	generator := NewHelpGenerator(app)
	fmt.Printf("%s", generator.GenerateCommandHelp(cmd))
	return nil
}

// GenerateHelp generates the main help text for the application.
// This method provides a simple help text format and can be used
// when you need to get the help content as a string instead of
// printing it directly.
func (app *App) GenerateHelp() string {
	var sb strings.Builder

	// Header
	if app.description != "" {
		sb.WriteString(app.description + "\n\n")
	}

	sb.WriteString(fmt.Sprintf("Usage: %s [command] [flags]\n\n", app.name))

	// Available commands
	if len(app.commands) > 0 {
		sb.WriteString("Available Commands:\n")
		for name, cmd := range app.commands {
			sb.WriteString(fmt.Sprintf("  %-12s %s\n", name, cmd.Description()))
		}
		sb.WriteString("\n")
	}

	// Global flags
	sb.WriteString("Global Flags:\n")
	sb.WriteString("  -h, --help      Show help\n")
	if app.version != "" {
		sb.WriteString("  -v, --version   Show version\n")
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Use \"%s help [command]\" for more information about a command.\n", app.name))

	return sb.String()
}

// GetCommands returns a copy of the commands map for introspection.
func (app *App) GetCommands() map[string]*Command {
	commands := make(map[string]*Command)
	for name, cmd := range app.commands {
		commands[name] = cmd
	}
	return commands
}

// Name returns the application name.
func (app *App) Name() string {
	return app.name
}

// Version returns the application version.
func (app *App) Version() string {
	return app.version
}

// Description returns the application description.
func (app *App) Description() string {
	return app.description
}
