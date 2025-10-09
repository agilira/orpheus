// context.go: context represents the execution context for a command
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	flashflags "github.com/agilira/flash-flags"
)

// Context represents the execution context for a command during execution.
// It provides access to parsed arguments, flags, and application state.
type Context struct {
	// App is the application instance
	App *App

	// Command is the currently executing command
	Command *Command

	// Args are the raw arguments passed to the command (after command name)
	Args []string

	// Flags is the parsed flag set for this command
	Flags *flashflags.FlagSet

	// GlobalFlags is the parsed global flag set
	GlobalFlags *flashflags.FlagSet
}

// GetArg returns the argument at the specified index.
// Returns empty string if index is out of bounds.
func (ctx *Context) GetArg(index int) string {
	if index < 0 || index >= len(ctx.Args) {
		return ""
	}
	return ctx.Args[index]
}

// ArgCount returns the number of arguments.
func (ctx *Context) ArgCount() int {
	return len(ctx.Args)
}

// GetFlag returns the value of a flag as interface{}.
func (ctx *Context) GetFlag(name string) interface{} {
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
func (ctx *Context) GetGlobalFlag(name string) interface{} {
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
func (ctx *Context) Logger() Logger {
	if ctx.App != nil {
		return ctx.App.logger
	}
	return nil
}

// AuditLogger returns the configured audit logger, or nil if not set.
func (ctx *Context) AuditLogger() AuditLogger {
	if ctx.App != nil {
		return ctx.App.auditLogger
	}
	return nil
}

// Tracer returns the configured tracer, or nil if not set.
func (ctx *Context) Tracer() Tracer {
	if ctx.App != nil {
		return ctx.App.tracer
	}
	return nil
}

// MetricsCollector returns the configured metrics collector, or nil if not set.
func (ctx *Context) MetricsCollector() MetricsCollector {
	if ctx.App != nil {
		return ctx.App.metricsCollector
	}
	return nil
}
