// completion.go: auto-completion bash/zsh
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"
	"sort"
	"strings"

	flashflags "github.com/agilira/flash-flags"
)

// GenerateCompletion generates bash completion script for the application.
func (app *App) GenerateCompletion(shell string) string {
	switch shell {
	case "bash":
		return app.generateBashCompletion()
	case "zsh":
		return app.generateZshCompletion()
	case "fish":
		return app.generateFishCompletion()
	default:
		return app.generateBashCompletion() // Default to bash
	}
}

// Complete provides completion suggestions for the current input.
func (app *App) Complete(args []string, position int) *CompletionResult {
	if len(args) == 0 || position == 0 {
		return app.completeCommands("")
	}

	// If we're completing the first argument, it's a command
	if position == 1 {
		return app.completeCommands(args[0])
	}

	// We're completing arguments or flags for a command
	cmdName := args[0]
	cmd, exists := app.commands[cmdName]
	if !exists {
		return &CompletionResult{Suggestions: []string{}}
	}

	currentWord := ""
	if position < len(args) {
		currentWord = args[position]
	} else if position == len(args) && len(args) > 1 {
		// We're at the end, check the last argument
		currentWord = args[len(args)-1]
	}

	// If current word starts with -, complete flags
	if strings.HasPrefix(currentWord, "-") {
		return app.completeFlags(cmd, currentWord)
	}

	// Complete arguments for the command
	req := &CompletionRequest{
		Type:        CompletionArgs,
		CurrentWord: currentWord,
		Command:     cmdName,
		Args:        args[1:],
		Position:    position - 1,
	}

	// Use custom completion handler if available
	if handler := cmd.completionHandler; handler != nil {
		return handler(req)
	}

	// Default: no suggestions
	return &CompletionResult{Suggestions: []string{}}
}

// completeCommands provides completion for command names.
func (app *App) completeCommands(partial string) *CompletionResult {
	var suggestions []string

	for name := range app.commands {
		if strings.HasPrefix(name, partial) {
			suggestions = append(suggestions, name)
		}
	}

	// Add built-in commands
	if strings.HasPrefix("help", partial) {
		suggestions = append(suggestions, "help")
	}

	sort.Strings(suggestions)
	return &CompletionResult{Suggestions: suggestions}
}

// completeFlags provides completion for command flags.
func (app *App) completeFlags(cmd *Command, partial string) *CompletionResult {
	var suggestions []string

	// Add global flags
	suggestions = append(suggestions, "--help", "-h")
	if app.version != "" {
		suggestions = append(suggestions, "--version", "-v")
	}

	// Add custom global flags
	if app.globalFlags != nil {
		app.globalFlags.VisitAll(func(flag *flashflags.Flag) {
			flagName := "--" + flag.Name()
			if strings.HasPrefix(flagName, partial) {
				suggestions = append(suggestions, flagName)
			}
		})
	}

	// Add command-specific flags
	if cmd.Flags() != nil {
		cmd.Flags().VisitAll(func(flag *flashflags.Flag) {
			flagName := "--" + flag.Name()
			if strings.HasPrefix(flagName, partial) {
				suggestions = append(suggestions, flagName)
			}
		})
	}

	// Filter by partial match and remove duplicates
	var filtered []string
	seen := make(map[string]bool)
	for _, flag := range suggestions {
		if strings.HasPrefix(flag, partial) && !seen[flag] {
			filtered = append(filtered, flag)
			seen[flag] = true
		}
	}

	sort.Strings(filtered)
	return &CompletionResult{
		Suggestions: filtered,
		Directive:   CompletionNoFiles,
	}
}

// generateBashCompletion generates a bash completion script.
func (app *App) generateBashCompletion() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`# Bash completion for %s
_%s_completion() {
    local cur prev words cword
    _init_completion || return

    case $cword in
        1)
            # Complete command names
            COMPREPLY=($(compgen -W "%s help" -- "$cur"))
            return 0
            ;;
        *)
            # Complete based on the command
            case ${words[1]} in
`, app.name, app.name, app.getCommandNames()))

	// Add completion for each command
	for name := range app.commands {
		sb.WriteString(fmt.Sprintf(`                %s)
                    COMPREPLY=($(compgen -W "--help -h" -- "$cur"))
                    return 0
                    ;;
`, name))
	}

	sb.WriteString(`                help)
                    COMPREPLY=($(compgen -W "`)
	sb.WriteString(app.getCommandNames())
	sb.WriteString(`" -- "$cur"))
                    return 0
                    ;;
            esac
            ;;
    esac
}

complete -F _`)
	sb.WriteString(app.name)
	sb.WriteString("_completion ")
	sb.WriteString(app.name)
	sb.WriteString("\n")

	return sb.String()
}

// generateZshCompletion generates a zsh completion script.
func (app *App) generateZshCompletion() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`#compdef %s

_%s() {
    local context curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments \
        '1: :->commands' \
        '*: :->args'

    case $state in
        commands)
            _describe 'commands' '(
`, app.name, app.name))

	// Add command descriptions for zsh
	for name, cmd := range app.commands {
		sb.WriteString(fmt.Sprintf("                %s:'%s'\n", name, cmd.Description()))
	}
	sb.WriteString("                help:'Show help for commands'\n")

	sb.WriteString(`            )'
            ;;
        args)
            case $words[2] in
                help)
                    _describe 'commands' '(
`)

	for name, cmd := range app.commands {
		sb.WriteString(fmt.Sprintf("                        %s:'%s'\n", name, cmd.Description()))
	}

	sb.WriteString(`                    )'
                    ;;
                *)
                    _arguments \
                        '--help[Show help]' \
                        '-h[Show help]'
                    ;;
            esac
            ;;
    esac
}

_`)
	sb.WriteString(app.name)
	sb.WriteString(" \"$@\"\n")

	return sb.String()
}

// generateFishCompletion generates a fish completion script.
func (app *App) generateFishCompletion() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Fish completion for %s\n\n", app.name))

	// Complete command names
	sb.WriteString(fmt.Sprintf("complete -c %s -f\n", app.name))

	// Add completions for each command
	for name, cmd := range app.commands {
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a %s -d '%s'\n",
			app.name, name, cmd.Description()))
	}

	// Add help command
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a help -d 'Show help for commands'\n",
		app.name))

	// Add global flags
	sb.WriteString(fmt.Sprintf("complete -c %s -s h -l help -d 'Show help'\n", app.name))
	if app.version != "" {
		sb.WriteString(fmt.Sprintf("complete -c %s -s v -l version -d 'Show version'\n", app.name))
	}

	// Add help completions for each command
	for name := range app.commands {
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from help' -a %s\n",
			app.name, name))
	}

	return sb.String()
}

// getCommandNames returns a space-separated list of command names.
func (app *App) getCommandNames() string {
	var names []string
	for name := range app.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, " ")
}

// AddCompletionCommand adds a built-in completion command to the app.
func (app *App) AddCompletionCommand() *App {
	app.Command("completion", "Generate shell completion scripts", func(ctx *Context) error {
		shell := "bash" // default
		if ctx.ArgCount() > 0 {
			shell = ctx.GetArg(0)
		}

		validShells := map[string]bool{
			"bash": true,
			"zsh":  true,
			"fish": true,
		}

		if !validShells[shell] {
			return ValidationError("completion", fmt.Sprintf("unsupported shell: %s (supported: bash, zsh, fish)", shell))
		}

		fmt.Print(app.GenerateCompletion(shell))
		return nil
	})

	return app
}
