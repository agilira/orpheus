// context.go: context represents the execution context for a command
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"

	flashflags "github.com/agilira/flash-flags"
	"github.com/agilira/orpheus/errs"
	"github.com/agilira/orpheus/observe"
	"github.com/agilira/orpheus/prompt"
	"github.com/agilira/orpheus/storage"
)

// Context represents the execution context for a command during execution.
// It provides access to parsed arguments, flags, and application state.
type Context struct {
	// App is the application instance
	App *App

	// Command is the currently executing command
	Command *Command

	// Args are the raw arguments passed to the command (after command name),
	// flags included. GetArg and ArgCount read the positional arguments.
	Args []string

	// Flags is the parsed flag set for this command
	Flags *flashflags.FlagSet

	// GlobalFlags is the parsed global flag set
	GlobalFlags *flashflags.FlagSet

	// ctx is the parent context for command execution.
	ctx context.Context

	// Storage provides access to the configured storage backend (optional)
	// Will be nil if storage is not configured for this application
	storage storage.Storage

	// prompter provides interactive terminal prompts (optional).
	// Will be nil if no prompter is configured (non-interactive mode).
	prompter prompt.Prompter
}

// Context returns the parent context for command execution.
// It falls back to context.Background when the context was built manually.
func (ctx *Context) Context() context.Context {
	if ctx.ctx != nil {
		return ctx.ctx
	}
	return context.Background()
}

// GetArg returns the positional argument at index, or "" when there is
// none. Flags and their values are not positional: in
// "app remote add -f origin URL", GetArg(0) is "origin".
//
// WHY positional: GetArg used to index ctx.Args, the raw arguments, so a
// flag written before the positionals was read as one. Every handler in the
// README and the examples called GetArg(0) for the first operand, and the
// gitlike example added a remote named "-f". ctx.Args keeps the raw view for
// the rare handler that wants it.
//
// A Context built outside the dispatch path has no parsed flags; ctx.Args
// is then the only view there is, and GetArg reads it.
func (ctx *Context) GetArg(index int) string {
	args := ctx.positionals()
	if index < 0 || index >= len(args) {
		return ""
	}
	return args[index]
}

// ArgCount returns the number of positional arguments. See GetArg.
func (ctx *Context) ArgCount() int {
	return len(ctx.positionals())
}

// positionals is the parser's positional arguments, or ctx.Args when no
// flags were parsed.
func (ctx *Context) positionals() []string {
	if ctx.Flags == nil {
		return ctx.Args
	}
	return ctx.Flags.Args()
}

// Positional returns the positional arguments after flag parsing, or nil
// when no flags have been parsed (a Context built outside the dispatch
// path). Since v1.9.0 GetArg and ArgCount read the same arguments;
// Positional, PositionalCount and GetPositional remain for the code that
// adopted them in v1.3.0.
func (ctx *Context) Positional() []string {
	if ctx.Flags == nil {
		return nil
	}
	return ctx.Flags.Args()
}

// PositionalCount returns the number of positional arguments after flag
// parsing. See Positional.
func (ctx *Context) PositionalCount() int {
	if ctx.Flags == nil {
		return 0
	}
	return len(ctx.Flags.Args())
}

// GetPositional returns the positional argument at index, or "" when there
// is none. See Positional.
func (ctx *Context) GetPositional(index int) string {
	if ctx.Flags == nil {
		return ""
	}
	args := ctx.Flags.Args()
	if index < 0 || index >= len(args) {
		return ""
	}
	return args[index]
}

// GetFlag returns the value of a flag as interface{}.
func (ctx *Context) GetFlag(name string) any {
	if ctx.Flags != nil {
		if flag := ctx.Flags.Lookup(name); flag != nil {
			return flag.Value()
		}
	}
	return nil
}

// GetFlagString returns a flag value as string.
func (ctx *Context) GetFlagString(name string) string {
	if ctx.Flags != nil {
		return ctx.Flags.GetString(name)
	}
	return ""
}

// GetFlagBool returns a flag value as bool.
func (ctx *Context) GetFlagBool(name string) bool {
	if ctx.Flags != nil {
		return ctx.Flags.GetBool(name)
	}
	return false
}

// GetFlagInt returns a flag value as int.
func (ctx *Context) GetFlagInt(name string) int {
	if ctx.Flags != nil {
		return ctx.Flags.GetInt(name)
	}
	return 0
}

// GetFlagFloat64 returns a flag value as float64.
func (ctx *Context) GetFlagFloat64(name string) float64 {
	if ctx.Flags != nil {
		return ctx.Flags.GetFloat64(name)
	}
	return 0.0
}

// GetFlagStringSlice returns a flag value as []string.
func (ctx *Context) GetFlagStringSlice(name string) []string {
	if ctx.Flags != nil {
		return ctx.Flags.GetStringSlice(name)
	}
	return []string{}
}

// FlagChanged returns whether the specified flag was set.
func (ctx *Context) FlagChanged(name string) bool {
	if ctx.Flags != nil {
		return ctx.Flags.Changed(name)
	}
	return false
}

// GetGlobalFlag returns the value of a global flag.
func (ctx *Context) GetGlobalFlag(name string) any {
	if ctx.GlobalFlags != nil {
		if flag := ctx.GlobalFlags.Lookup(name); flag != nil {
			return flag.Value()
		}
	}
	return nil
}

// GetGlobalFlagString returns a global flag value as string.
func (ctx *Context) GetGlobalFlagString(name string) string {
	if ctx.GlobalFlags != nil {
		return ctx.GlobalFlags.GetString(name)
	}
	return ""
}

// GetGlobalFlagBool returns a global flag value as bool.
func (ctx *Context) GetGlobalFlagBool(name string) bool {
	if ctx.GlobalFlags != nil {
		return ctx.GlobalFlags.GetBool(name)
	}
	return false
}

// GetGlobalFlagInt returns a global flag value as int.
func (ctx *Context) GetGlobalFlagInt(name string) int {
	if ctx.GlobalFlags != nil {
		return ctx.GlobalFlags.GetInt(name)
	}
	return 0
}

// GlobalFlagChanged returns whether the specified global flag was set.
func (ctx *Context) GlobalFlagChanged(name string) bool {
	if ctx.GlobalFlags != nil {
		return ctx.GlobalFlags.Changed(name)
	}
	return false
}

// Logger returns the configured logger, or nil if not set.
func (ctx *Context) Logger() observe.Logger {
	if ctx.App != nil {
		return ctx.App.logger
	}
	return nil
}

// AuditLogger returns the configured audit logger, or nil if not set.
func (ctx *Context) AuditLogger() observe.AuditLogger {
	if ctx.App != nil {
		return ctx.App.auditLogger
	}
	return nil
}

// Tracer returns the configured tracer, or nil if not set.
func (ctx *Context) Tracer() observe.Tracer {
	if ctx.App != nil {
		return ctx.App.tracer
	}
	return nil
}

// MetricsCollector returns the configured metrics collector, or nil if not set.
func (ctx *Context) MetricsCollector() observe.MetricsCollector {
	if ctx.App != nil {
		return ctx.App.metricsCollector
	}
	return nil
}

// Storage returns the configured storage backend, or nil if not configured.
// This provides zero-overhead access - if storage is not used, there's no performance impact.
func (ctx *Context) Storage() storage.Storage {
	return ctx.storage
}

// RequireStorage returns the storage backend or an error if not configured.
// Use this when storage is required for the operation to proceed.
func (ctx *Context) RequireStorage() (storage.Storage, error) {
	if ctx.storage == nil {
		return nil, storage.ValidationError("require", "storage not configured")
	}
	return ctx.storage, nil
}

// SetStorage configures the storage backend for this context.
// This is typically called during application initialization.
func (ctx *Context) SetStorage(storage storage.Storage) {
	ctx.storage = storage
}

// Prompter returns the configured prompter, or nil if not configured.
// Handlers should check for nil before using:
//
//	p := ctx.Prompter()
//	if p == nil {
//	    return fmt.Errorf("interactive mode required")
//	}
func (ctx *Context) Prompter() prompt.Prompter {
	return ctx.prompter
}

// RequirePrompter returns the prompter or an error if not configured.
// WHY: commands that need user input should fail clearly when running
// in non-interactive mode (e.g., cron, CI, piped input without prompter).
func (ctx *Context) RequirePrompter() (prompt.Prompter, error) {
	if ctx.prompter == nil {
		return nil, errs.ValidationError("", "interactive prompts not available: no prompter configured")
	}
	return ctx.prompter, nil
}

// forSubcommand derives the execution context for a subcommand, keeping
// everything the parent context carries and overriding only what the descent
// changes: the command being executed and the arguments left for it.
//
// WHY a copy of the whole struct rather than a fresh literal: the previous
// code listed each field it wanted to keep, so every field added to Context
// afterwards was silently dropped at the first subcommand boundary. That is
// how the parent context.Context stopped reaching nested handlers -- signal
// cancellation installed by Run was invisible to every `app cmd subcmd`
// handler, with nothing to indicate it. Copying and then overriding makes
// inheritance the default, so a new field is carried without anyone
// remembering to add it here.
//
// Flags is deliberately cleared: it holds the PARENT command's parsed flag
// set, and parseAndExecute assigns the subcommand's own set once it parses.
// Leaving the parent's would let a handler read a flag that was never parsed
// for the command it belongs to.
func (ctx *Context) forSubcommand(subcmd *Command, args []string) *Context {
	child := *ctx
	child.Command = subcmd
	child.Args = args
	child.Flags = nil
	return &child
}
