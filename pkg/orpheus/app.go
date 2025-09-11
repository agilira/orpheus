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
	name        string
	description string
	version     string
	commands    map[string]*Command
	globalFlags *flashflags.FlagSet
	defaultCmd  string
	helpCommand *Command
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
	// Handle special cases
	if len(args) == 0 {
		if app.defaultCmd != "" {
			return app.runCommand(app.defaultCmd, []string{})
		}
		return app.helpHandler(&Context{App: app})
	}

	// Check for global help flag
	if args[0] == "--help" || args[0] == "-h" {
		return app.helpHandler(&Context{App: app})
	}

	// Check for version flag
	if args[0] == "--version" || args[0] == "-v" {
		if app.version != "" {
			fmt.Printf("%s version %s\n", app.name, app.version)
		} else {
			fmt.Printf("%s (no version set)\n", app.name)
		}
		return nil
	}

	// Parse global flags first
	globalArgs, cmdArgs := app.splitGlobalArgs(args)
	if err := app.globalFlags.Parse(globalArgs); err != nil {
		return ValidationError("", "global flag parsing failed: "+err.Error())
	}

	// Get command name
	if len(cmdArgs) == 0 {
		if app.defaultCmd != "" {
			return app.runCommand(app.defaultCmd, []string{})
		}
		return app.helpHandler(&Context{App: app})
	}

	cmdName := cmdArgs[0]
	cmdArgs = cmdArgs[1:] // Remove command name from args

	// Handle built-in help command
	if cmdName == "help" {
		if len(cmdArgs) > 0 {
			return app.showCommandHelp(cmdArgs[0])
		}
		return app.helpHandler(&Context{App: app})
	}

	return app.runCommand(cmdName, cmdArgs)
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

		// Check if this is a boolean global flag
		isBoolFlag := false
		if strings.HasPrefix(arg, "--") {
			flagName := arg[2:]
			if eqPos := strings.IndexByte(flagName, '='); eqPos != -1 {
				flagName = flagName[:eqPos]
			}
			if flag := app.globalFlags.Lookup(flagName); flag != nil && flag.Type() == "bool" {
				isBoolFlag = true
			}
		} else if len(arg) == 2 && arg[0] == '-' {
			// Check short flags - we'd need to map them to long names
			// For now, treat common boolean short flags as boolean
			shortKey := string(arg[1])
			if shortKey == "v" || shortKey == "h" || shortKey == "d" {
				isBoolFlag = true
			}
		}

		if isBoolFlag || strings.Contains(arg, "=") {
			// Boolean flag or flag with embedded value
			globalArgs = append(globalArgs, arg)
		} else {
			// Flag that might need a value
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				globalArgs = append(globalArgs, arg, args[i+1])
				i++ // Skip the value
			} else {
				globalArgs = append(globalArgs, arg)
			}
		}
	}

	// Everything from the command onwards is command args
	cmdArgs = args[i:]
	return globalArgs, cmdArgs
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
