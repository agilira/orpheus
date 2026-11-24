// facade_core.go: the App, Command and Context API of
// github.com/agilira/orpheus under its v1.11.0 import path
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import "github.com/agilira/orpheus"

// Application, command and completion types, see package
// github.com/agilira/orpheus.
type (
	App                 = orpheus.App
	Command             = orpheus.Command
	CommandHandler      = orpheus.CommandHandler
	Context             = orpheus.Context
	HelpGenerator       = orpheus.HelpGenerator
	CompletionType      = orpheus.CompletionType
	CompletionRequest   = orpheus.CompletionRequest
	CompletionResult    = orpheus.CompletionResult
	CompletionDirective = orpheus.CompletionDirective
	CompletionHandler   = orpheus.CompletionHandler
)

// Completion types and directives, see package github.com/agilira/orpheus.
const (
	CompletionCommands = orpheus.CompletionCommands
	CompletionFlags    = orpheus.CompletionFlags
	CompletionArgs     = orpheus.CompletionArgs
	CompletionDefault  = orpheus.CompletionDefault
	CompletionNoSpace  = orpheus.CompletionNoSpace
	CompletionNoFiles  = orpheus.CompletionNoFiles
)

// New calls orpheus.New.
func New(name string) *App { return orpheus.New(name) }

// NewCommand calls orpheus.NewCommand.
func NewCommand(name, description string) *Command { return orpheus.NewCommand(name, description) }

// NewHelpGenerator calls orpheus.NewHelpGenerator.
func NewHelpGenerator(app *App) *HelpGenerator { return orpheus.NewHelpGenerator(app) }
