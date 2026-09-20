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
//
// It suggests a built-in spelling only while the application has left it free,
// for the same reason the parser and the help generator do: suggesting -v for
// the version flag to an application that registered -v for verbosity offers a
// completion the program will not honour. Registered flags contribute both
// their long name and their short key, so a shorthand is discoverable by
// pressing tab, not only by reading the source.
func (app *App) completeFlags(cmd *Command, partial string) *CompletionResult {
	var suggestions []string

	// Built-in flags, in the spellings still free at the level that handles them
	suggestions = append(suggestions, builtinFlagSuggestions(app.globalFlags, "help", "h")...)
	if app.version != "" {
		suggestions = append(suggestions, builtinFlagSuggestions(app.globalFlags, "version", "v")...)
	}

	// Add custom global flags
	suggestions = append(suggestions, flagSuggestions(app.globalFlags)...)

	// Add command-specific flags
	suggestions = append(suggestions, flagSuggestions(cmd.Flags())...)

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

// builtinFlagSuggestions returns the spellings of a built-in flag that fs has
// not claimed. Both spellings claimed means the built-in is unreachable, and
// nothing is suggested.
func builtinFlagSuggestions(fs *flashflags.FlagSet, longName, shortKey string) []string {
	var out []string
	if !flagNameTaken(fs, longName) {
		out = append(out, "--"+longName)
	}
	if !shortKeyTaken(fs, shortKey) {
		out = append(out, "-"+shortKey)
	}
	return out
}

// flagSuggestions returns every spelling registered in fs: the long name, plus
// the short key where one was given.
func flagSuggestions(fs *flashflags.FlagSet) []string {
	if fs == nil {
		return nil
	}
	var out []string
	fs.VisitAll(func(flag *flashflags.Flag) {
		out = append(out, "--"+flag.Name())
		if short := flag.ShortKey(); short != "" {
			out = append(out, "-"+short)
		}
	})
	return out
}

// sortedCommands returns the application's commands in name order.
//
// WHY it exists: the generators used to range over app.commands directly, and
// Go randomises map iteration, so every invocation emitted the command blocks
// in a different order. A completion script is normally generated once and
// written to a file -- `myapp completion bash > /etc/bash_completion.d/myapp`
// -- where an output that changes on every run shows up as a spurious diff on
// every rebuild, and defeats any checksum over it.
func (app *App) sortedCommands() []*Command {
	names := make([]string, 0, len(app.commands))
	for name := range app.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	commands := make([]*Command, 0, len(names))
	for _, name := range names {
		commands = append(commands, app.commands[name])
	}
	return commands
}

// quotePOSIXSingle escapes s for use inside a single-quoted POSIX shell string,
// as bash and zsh read one: the quote is closed, an escaped quote emitted, and
// the quote reopened.
//
// WHY: a description is written by the application author, not a user, but an
// apostrophe is ordinary English -- "Show user's config" used to close the
// string early and emit a syntactically broken completion script, which the
// shell reports far from its cause.
func quotePOSIXSingle(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

// quoteFishSingle escapes s for use inside a single-quoted fish string, where
// only the backslash and the quote itself are escapable.
func quoteFishSingle(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, "'", `\'`)
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
	for _, cmd := range app.sortedCommands() {
		sb.WriteString(fmt.Sprintf(`                %s)
                    COMPREPLY=($(compgen -W "--help -h" -- "$cur"))
                    return 0
                    ;;
`, cmd.Name()))
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
	for _, cmd := range app.sortedCommands() {
		sb.WriteString(fmt.Sprintf("                %s:'%s'\n", cmd.Name(), quotePOSIXSingle(cmd.Description())))
	}
	sb.WriteString("                help:'Show help for commands'\n")

	sb.WriteString(`            )'
            ;;
        args)
            case $words[2] in
                help)
                    _describe 'commands' '(
`)

	for _, cmd := range app.sortedCommands() {
		sb.WriteString(fmt.Sprintf("                        %s:'%s'\n", cmd.Name(), quotePOSIXSingle(cmd.Description())))
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
	for _, cmd := range app.sortedCommands() {
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a %s -d '%s'\n",
			app.name, cmd.Name(), quoteFishSingle(cmd.Description())))
	}

	// Add help command
	sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a help -d 'Show help for commands'\n",
		app.name))

	// Add global flags
	sb.WriteString(app.fishBuiltinFlag("help", "h", "Show help"))
	if app.version != "" {
		sb.WriteString(app.fishBuiltinFlag("version", "v", "Show version"))
	}

	// Add help completions for each command
	for _, cmd := range app.sortedCommands() {
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from help' -a %s\n",
			app.name, cmd.Name()))
	}

	return sb.String()
}

// fishBuiltinFlag emits the fish completion for a built-in flag, naming only
// the spellings the application has left free. Both claimed emits nothing.
func (app *App) fishBuiltinFlag(longName, shortKey, description string) string {
	var spelling string
	longFree := !flagNameTaken(app.globalFlags, longName)
	shortFree := !shortKeyTaken(app.globalFlags, shortKey)

	switch {
	case longFree && shortFree:
		spelling = fmt.Sprintf("-s %s -l %s", shortKey, longName)
	case longFree:
		spelling = "-l " + longName
	case shortFree:
		spelling = "-s " + shortKey
	default:
		return ""
	}

	return fmt.Sprintf("complete -c %s %s -d '%s'\n", app.name, spelling, quoteFishSingle(description))
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
