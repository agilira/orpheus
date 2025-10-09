// Basic Orpheus CLI Example
//
// This example demonstrates the fundamental features of Orpheus:
// - Simple command creation
// - Global flags
// - Argument handling
// - Error handling
// - Command completion
// - Help generation
//
// Usage examples:
//   basic greet
//   basic greet Alice
//   basic --verbose greet Bob
//   basic echo hello world
//   basic deploy production
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// createBasicApp creates and configures the basic example application
func createBasicApp() *orpheus.App {
	// Create a new Orpheus application
	app := orpheus.New("basic").
		SetDescription("Basic Orpheus CLI Framework Example").
		SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output").
		AddGlobalFlag("config", "c", "", "Configuration file path")

	// Add a simple greeting command
	app.Command("greet", "Greet someone", func(ctx *orpheus.Context) error {
		name := "World"
		if ctx.ArgCount() > 0 {
			name = ctx.GetArg(0)
		}

		if ctx.GetGlobalFlagBool("verbose") {
			fmt.Printf("Greeting %s with verbose output enabled\n", name)
		}

		fmt.Printf("Hello, %s!\n", name)
		return nil
	})

	// Add a more complex echo command
	echoCmd := orpheus.NewCommand("echo", "Echo back the arguments").
		SetHandler(func(ctx *orpheus.Context) error {
			if ctx.ArgCount() == 0 {
				return orpheus.ValidationError("echo", "no arguments provided")
			}

			for i := 0; i < ctx.ArgCount(); i++ {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(ctx.GetArg(i))
			}
			fmt.Println()
			return nil
		})

	// Add a deploy command with custom completion
	deployCmd := orpheus.NewCommand("deploy", "Deploy to environment").
		SetHandler(func(ctx *orpheus.Context) error {
			if ctx.ArgCount() == 0 {
				return orpheus.ValidationError("deploy", "environment required")
			}

			env := ctx.GetArg(0)
			fmt.Printf("Deploying to %s environment...\n", env)
			return nil
		}).
		SetCompletionHandler(func(req *orpheus.CompletionRequest) *orpheus.CompletionResult {
			if req.Type == orpheus.CompletionArgs && req.Position == 0 {
				return &orpheus.CompletionResult{
					Suggestions: []string{"production", "staging", "development"},
				}
			}
			return &orpheus.CompletionResult{Suggestions: []string{}}
		})

	app.AddCommand(echoCmd).
		AddCommand(deployCmd)

	// Add completion command
	app.AddCompletionCommand()

	// Set default command
	app.SetDefaultCommand("greet")

	return app
}

func main() {
	app := createBasicApp()

	// Run the application
	if err := app.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
