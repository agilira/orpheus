// benchmark_test.go: benchmarks with various popular CLI frameworks
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package benchmarks

import (
	"flag"
	"os"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
	"github.com/alecthomas/kingpin/v2"
	"github.com/spf13/cobra"
	"github.com/urfave/cli/v2"
)

// Benchmark scenario: Parse command with 3 flags and execute
// This simulates a typical CLI operation

// Orpheus implementation
func BenchmarkOrpheus(b *testing.B) {
	app := orpheus.New("benchmark")
	cmd := orpheus.NewCommand("deploy", "Deploy application")
	cmd.AddFlag("env", "e", "prod", "Environment")
	cmd.AddBoolFlag("verbose", "v", false, "Verbose output")
	cmd.AddFlag("timeout", "t", "30", "Timeout in seconds")
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		// Simulate some work
		_ = ctx.GetFlagString("env")
		_ = ctx.GetFlagBool("verbose")
		_ = ctx.GetFlagString("timeout")
		return nil
	})
	app.AddCommand(cmd)

	args := []string{"deploy", "--env", "staging", "--verbose", "--timeout", "60"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := app.Run(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Cobra implementation
func BenchmarkCobra(b *testing.B) {
	var env, timeout string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy application",
		Run: func(cmd *cobra.Command, args []string) {
			// Simulate some work
			_ = env
			_ = verbose
			_ = timeout
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "prod", "Environment")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().StringVarP(&timeout, "timeout", "t", "30", "Timeout in seconds")

	rootCmd := &cobra.Command{Use: "benchmark"}
	rootCmd.AddCommand(cmd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rootCmd.SetArgs([]string{"deploy", "--env", "staging", "--verbose", "--timeout", "60"})
		err := rootCmd.Execute()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Urfave/cli implementation
func BenchmarkUrfaveCli(b *testing.B) {
	app := &cli.App{
		Name: "benchmark",
		Commands: []*cli.Command{
			{
				Name:  "deploy",
				Usage: "Deploy application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "env",
						Aliases: []string{"e"},
						Value:   "prod",
						Usage:   "Environment",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Verbose output",
					},
					&cli.StringFlag{
						Name:    "timeout",
						Aliases: []string{"t"},
						Value:   "30",
						Usage:   "Timeout in seconds",
					},
				},
				Action: func(c *cli.Context) error {
					// Simulate some work
					_ = c.String("env")
					_ = c.Bool("verbose")
					_ = c.String("timeout")
					return nil
				},
			},
		},
	}

	args := []string{"benchmark", "deploy", "--env", "staging", "--verbose", "--timeout", "60"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := app.Run(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Kingpin implementation
func BenchmarkKingpin(b *testing.B) {
	app := kingpin.New("benchmark", "Benchmark application")
	deploy := app.Command("deploy", "Deploy application")
	env := deploy.Flag("env", "Environment").Short('e').Default("prod").String()
	verbose := deploy.Flag("verbose", "Verbose output").Short('v').Bool()
	timeout := deploy.Flag("timeout", "Timeout in seconds").Short('t').Default("30").String()

	args := []string{"deploy", "--env", "staging", "--verbose", "--timeout", "60"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd, err := app.Parse(args)
		if err != nil {
			b.Fatal(err)
		}
		if cmd == deploy.FullCommand() {
			// Simulate some work
			_ = *env
			_ = *verbose
			_ = *timeout
		}
	}
}

// Standard library flag (baseline)
func BenchmarkStdFlag(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset flag state
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		env := flag.String("env", "prod", "Environment")
		verbose := flag.Bool("verbose", false, "Verbose output")
		timeout := flag.String("timeout", "30", "Timeout in seconds")

		args := []string{"-env", "staging", "-verbose", "-timeout", "60"}
		err := flag.CommandLine.Parse(args)
		if err != nil {
			b.Fatal(err)
		}

		// Simulate some work
		_ = *env
		_ = *verbose
		_ = *timeout
	}
}
