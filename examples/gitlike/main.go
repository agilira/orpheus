// Git-like CLI Demo with Subcommands
//
// This example demonstrates Orpheus's native subcommand support,
// showing a git-like interface with nested commands such as:
// - remote add/remove/list
// - config set/get/list
// - status
//
// Usage examples:
//   gitlike remote add origin https://github.com/user/repo.git
//   gitlike remote list
//   gitlike config set user.name "John Doe"
//   gitlike config get user.email
//   gitlike status
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// Data structures for persistence
type GitConfig struct {
	Remotes map[string]string `json:"remotes"`
	Config  map[string]string `json:"config"`
}

// Default configuration
var defaultGitConfig = GitConfig{
	Remotes: map[string]string{
		"origin": "https://github.com/agilira/orpheus.git",
	},
	Config: map[string]string{
		"user.name":  "Developer",
		"user.email": "dev@example.com",
	},
}

// Global data store
var gitData GitConfig

// getConfigFile returns the path to the config file
func getConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gitlike-config.json"
	}
	return filepath.Join(home, ".gitlike-config.json")
}

// isValidConfigPath validates the config file path for security
func isValidConfigPath(path string) bool {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)

	// Only allow files in home directory or current directory
	if strings.Contains(cleanPath, "..") {
		return false
	}

	// Must be a .json file
	if !strings.HasSuffix(cleanPath, ".json") {
		return false
	}

	return true
}

// loadConfig loads configuration from file
func loadConfig() {
	configFile := getConfigFile()

	// Validate config file path for security
	if !isValidConfigPath(configFile) {
		gitData = defaultGitConfig
		return
	}

	// #nosec G304 - path is validated above
	data, err := os.ReadFile(configFile)
	if err != nil {
		// File doesn't exist, use defaults
		gitData = defaultGitConfig
		return
	}

	err = json.Unmarshal(data, &gitData)
	if err != nil {
		// Invalid JSON, use defaults
		gitData = defaultGitConfig
		return
	}
}

// saveConfig saves configuration to file
func saveConfig() error {
	configFile := getConfigFile()

	data, err := json.MarshalIndent(gitData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0600)
}

func main() {
	// Load configuration at startup
	loadConfig()

	app := orpheus.New("gitlike").
		SetDescription("A Git-like CLI demonstration with native subcommands").
		SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output").
		AddGlobalFlag("config-file", "c", "~/.gitconfig", "Configuration file path")

	// Setup remote command with subcommands
	setupRemoteCommand(app)

	// Setup config command with subcommands
	setupConfigCommand(app)

	// Setup status command (simple command without subcommands)
	setupStatusCommand(app)

	// Setup branch command with subcommands
	setupBranchCommand(app)

	// Add completion support
	app.AddCompletionCommand()

	// Run the application with enhanced error handling
	if err := app.Run(os.Args[1:]); err != nil {
		if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
			log.Printf("Error: %s", orpheusErr.UserMessage())
			if orpheusErr.IsRetryable() {
				log.Printf("This operation can be retried")
			}
			os.Exit(orpheusErr.ExitCode())
		}
		log.Fatal(err)
	}
}

// setupRemoteCommand demonstrates git-style remote management
func setupRemoteCommand(app *orpheus.App) {
	remote := orpheus.NewCommand("remote", "Manage remote repositories")

	// remote add <name> <url>
	remote.AddSubcommand(
		orpheus.NewCommand("add", "Add a new remote repository").
			SetHandler(handleRemoteAdd).
			AddFlag("tags", "t", "", "Import tags from remote").
			AddBoolFlag("fetch", "f", false, "Fetch from remote after adding").
			SetUsage("add <name> <url>").
			AddExample("gitlike remote add origin https://github.com/user/repo.git").
			AddExample("gitlike remote add --fetch upstream https://github.com/upstream/repo.git"),
	)

	// remote remove <name>
	remote.AddSubcommand(
		orpheus.NewCommand("remove", "Remove a remote repository").
			SetHandler(handleRemoteRemove).
			SetUsage("remove <name>").
			AddExample("gitlike remote remove origin"),
	)

	// remote list
	remote.AddSubcommand(
		orpheus.NewCommand("list", "List all remote repositories").
			SetHandler(handleRemoteList).
			AddBoolFlag("verbose", "v", false, "Show URLs for remotes").
			AddExample("gitlike remote list").
			AddExample("gitlike remote list --verbose"),
	)

	// remote show <name>
	remote.AddSubcommand(
		orpheus.NewCommand("show", "Show information about a remote").
			SetHandler(handleRemoteShow).
			SetUsage("show <name>").
			AddExample("gitlike remote show origin"),
	)

	app.AddCommand(remote)
}

// setupConfigCommand demonstrates configuration management
func setupConfigCommand(app *orpheus.App) {
	configCmd := orpheus.NewCommand("config", "Manage configuration settings")

	// config set <key> <value>
	configCmd.AddSubcommand(
		orpheus.NewCommand("set", "Set a configuration value").
			SetHandler(handleConfigSet).
			AddBoolFlag("global", "g", false, "Set configuration globally").
			SetUsage("set <key> <value>").
			AddExample("gitlike config set user.name \"John Doe\"").
			AddExample("gitlike config set --global user.email john@example.com"),
	)

	// config get <key>
	configCmd.AddSubcommand(
		orpheus.NewCommand("get", "Get a configuration value").
			SetHandler(handleConfigGet).
			SetUsage("get <key>").
			AddExample("gitlike config get user.name").
			AddExample("gitlike config get user.email"),
	)

	// config list
	configCmd.AddSubcommand(
		orpheus.NewCommand("list", "List all configuration values").
			SetHandler(handleConfigList).
			AddBoolFlag("show-origin", "s", false, "Show origin of config values").
			AddExample("gitlike config list").
			AddExample("gitlike config list --show-origin"),
	)

	// config unset <key>
	configCmd.AddSubcommand(
		orpheus.NewCommand("unset", "Remove a configuration value").
			SetHandler(handleConfigUnset).
			SetUsage("unset <key>").
			AddExample("gitlike config unset user.nickname"),
	)

	app.AddCommand(configCmd)
}

// setupStatusCommand demonstrates a simple command without subcommands
func setupStatusCommand(app *orpheus.App) {
	status := orpheus.NewCommand("status", "Show the working tree status").
		SetHandler(handleStatus).
		AddBoolFlag("short", "s", false, "Give the output in short format").
		AddBoolFlag("branch", "b", false, "Show branch information").
		AddExample("gitlike status").
		AddExample("gitlike status --short")

	app.AddCommand(status)
}

// setupBranchCommand demonstrates nested subcommands
func setupBranchCommand(app *orpheus.App) {
	branch := orpheus.NewCommand("branch", "Manage branches")

	// branch list
	branch.AddSubcommand(
		orpheus.NewCommand("list", "List branches").
			SetHandler(handleBranchList).
			AddBoolFlag("all", "a", false, "List both remote and local branches").
			AddBoolFlag("remote", "r", false, "List remote branches only"),
	)

	// branch create <name>
	branch.AddSubcommand(
		orpheus.NewCommand("create", "Create a new branch").
			SetHandler(handleBranchCreate).
			SetUsage("create <name>").
			AddFlag("from", "f", "HEAD", "Create branch from this commit").
			AddExample("gitlike branch create feature/new-feature").
			AddExample("gitlike branch create --from develop hotfix/critical-fix"),
	)

	// branch delete <name>
	branch.AddSubcommand(
		orpheus.NewCommand("delete", "Delete a branch").
			SetHandler(handleBranchDelete).
			SetUsage("delete <name>").
			AddBoolFlag("force", "f", false, "Force delete branch").
			AddExample("gitlike branch delete feature/old-feature").
			AddExample("gitlike branch delete --force experimental"),
	)

	app.AddCommand(branch)
}

// Remote command handlers
func handleRemoteAdd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 2 {
		return orpheus.ValidationError("remote add", "requires name and URL: remote add <name> <url>").
			WithUserMessage("Please provide both remote name and URL").
			WithContext("usage", "gitlike remote add <name> <url>").
			WithContext("example", "gitlike remote add origin https://github.com/user/repo.git")
	}

	name := ctx.GetArg(0)
	url := ctx.GetArg(1)

	if _, exists := gitData.Remotes[name]; exists {
		return orpheus.ValidationError("remote add", fmt.Sprintf("remote '%s' already exists", name)).
			WithUserMessage(fmt.Sprintf("A remote named '%s' is already configured", name)).
			WithContext("existing_url", gitData.Remotes[name]).
			WithContext("attempted_url", url).
			WithContext("suggestion", "use 'remote remove' first or choose a different name")
	}

	gitData.Remotes[name] = url

	verbose := ctx.GetGlobalFlagBool("verbose")
	if verbose {
		fmt.Printf("Added remote '%s' with URL: %s\n", name, url)
	} else {
		fmt.Printf("Added remote: %s\n", name)
	}

	if ctx.GetFlagBool("fetch") {
		fmt.Printf("Fetching from %s...\n", name)
	}

	// Save changes with enhanced error handling
	if err := saveConfig(); err != nil {
		return orpheus.ExecutionError("remote add", fmt.Sprintf("failed to save configuration: %v", err)).
			WithUserMessage("Unable to save the remote configuration to disk").
			WithContext("remote_name", name).
			WithContext("remote_url", url).
			WithContext("config_file", "gitlike-config.json").
			AsRetryable().
			WithSeverity("warning")
	}

	return nil
}

func handleRemoteRemove(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return orpheus.ValidationError("remote remove", "requires remote name: remote remove <name>")
	}

	name := ctx.GetArg(0)

	if _, exists := gitData.Remotes[name]; !exists {
		return orpheus.ValidationError("remote remove", fmt.Sprintf("remote '%s' does not exist", name))
	}

	delete(gitData.Remotes, name)
	fmt.Printf("Removed remote: %s\n", name)

	// Save changes
	if err := saveConfig(); err != nil {
		fmt.Printf("Warning: could not save configuration: %v\n", err)
	}

	return nil
}

func handleRemoteList(ctx *orpheus.Context) error {
	if len(gitData.Remotes) == 0 {
		fmt.Println("No remotes configured")
		return nil
	}

	verbose := ctx.GetFlagBool("verbose")

	for name, url := range gitData.Remotes {
		if verbose {
			fmt.Printf("%s\t%s (fetch)\n", name, url)
			fmt.Printf("%s\t%s (push)\n", name, url)
		} else {
			fmt.Println(name)
		}
	}

	return nil
}

func handleRemoteShow(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return orpheus.ValidationError("remote show", "requires remote name: remote show <name>")
	}

	name := ctx.GetArg(0)

	url, exists := gitData.Remotes[name]
	if !exists {
		return orpheus.ValidationError("remote show", fmt.Sprintf("remote '%s' does not exist", name))
	}

	fmt.Printf("* remote %s\n", name)
	fmt.Printf("  Fetch URL: %s\n", url)
	fmt.Printf("  Push  URL: %s\n", url)
	fmt.Printf("  HEAD branch: main\n")
	fmt.Printf("  Remote branches:\n")
	fmt.Printf("    main tracked\n")
	fmt.Printf("    develop tracked\n")

	return nil
}

// Config command handlers
func handleConfigSet(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 2 {
		return orpheus.ValidationError("config set", "requires key and value: config set <key> <value>")
	}

	key := ctx.GetArg(0)
	value := ctx.GetArg(1)

	gitData.Config[key] = value

	scope := "local"
	if ctx.GetFlagBool("global") {
		scope = "global"
	}

	if ctx.GetGlobalFlagBool("verbose") {
		fmt.Printf("Set %s configuration: %s = %s\n", scope, key, value)
	} else {
		fmt.Printf("Set: %s = %s\n", key, value)
	}

	// Save changes
	if err := saveConfig(); err != nil {
		fmt.Printf("Warning: could not save configuration: %v\n", err)
	}

	return nil
}

func handleConfigGet(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return orpheus.ValidationError("config get", "requires configuration key: config get <key>")
	}

	key := ctx.GetArg(0)

	value, exists := gitData.Config[key]
	if !exists {
		return orpheus.NotFoundError("config get", fmt.Sprintf("configuration key '%s' not found", key)).
			WithUserMessage(fmt.Sprintf("No configuration found for '%s'", key)).
			WithContext("requested_key", key).
			WithContext("total_config_keys", len(gitData.Config)).
			WithContext("suggestion", "use 'config list' to see all available keys")
	}

	fmt.Println(value)
	return nil
}

func handleConfigList(ctx *orpheus.Context) error {
	if len(gitData.Config) == 0 {
		fmt.Println("No configuration values set")
		return nil
	}

	showOrigin := ctx.GetFlagBool("show-origin")

	for key, value := range gitData.Config {
		if showOrigin {
			fmt.Printf("file:~/.gitlike-config.json\t%s=%s\n", key, value)
		} else {
			fmt.Printf("%s=%s\n", key, value)
		}
	}

	return nil
}

func handleConfigUnset(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return orpheus.ValidationError("config unset", "requires configuration key: config unset <key>")
	}

	key := ctx.GetArg(0)

	if _, exists := gitData.Config[key]; !exists {
		return orpheus.ValidationError("config unset", fmt.Sprintf("configuration key '%s' not found", key))
	}

	delete(gitData.Config, key)
	fmt.Printf("Unset: %s\n", key)

	// Save changes
	if err := saveConfig(); err != nil {
		fmt.Printf("Warning: could not save configuration: %v\n", err)
	}

	return nil
}

// Status command handler
func handleStatus(ctx *orpheus.Context) error {
	short := ctx.GetFlagBool("short")
	showBranch := ctx.GetFlagBool("branch")

	if showBranch || !short {
		fmt.Println("On branch main")
	}

	if short {
		fmt.Println("?? untracked.txt")
		fmt.Println(" M modified.go")
		fmt.Println("A  added.md")
	} else {
		fmt.Println("Your branch is up to date with 'origin/main'.")
		fmt.Println()
		fmt.Println("Changes to be committed:")
		fmt.Println("  (use \"git restore --staged <file>...\" to unstage)")
		fmt.Println("\tnew file:   added.md")
		fmt.Println()
		fmt.Println("Changes not staged for commit:")
		fmt.Println("  (use \"git add <file>...\" to update what will be committed)")
		fmt.Println("  (use \"git restore <file>...\" to discard changes in working directory)")
		fmt.Println("\tmodified:   modified.go")
		fmt.Println()
		fmt.Println("Untracked files:")
		fmt.Println("  (use \"git add <file>...\" to include in what will be committed)")
		fmt.Println("\tuntracked.txt")
	}

	return nil
}

// Branch command handlers
func handleBranchList(ctx *orpheus.Context) error {
	all := ctx.GetFlagBool("all")
	remote := ctx.GetFlagBool("remote")

	// Local branches
	if !remote {
		fmt.Println("* main")
		fmt.Println("  develop")
		fmt.Println("  feature/new-cli")
	}

	// Remote branches
	if all || remote {
		if !remote {
			fmt.Println("  remotes/origin/HEAD -> origin/main")
		}
		fmt.Println("  remotes/origin/main")
		fmt.Println("  remotes/origin/develop")
		fmt.Println("  remotes/origin/release/v1.0")
	}

	return nil
}

func handleBranchCreate(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return orpheus.ValidationError("branch create", "requires branch name: branch create <name>")
	}

	name := ctx.GetArg(0)
	from := ctx.GetFlagString("from")

	if strings.Contains(name, " ") {
		return orpheus.ValidationError("branch create", "branch name cannot contain spaces")
	}

	fmt.Printf("Created branch '%s' from '%s'\n", name, from)
	return nil
}

func handleBranchDelete(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return orpheus.ValidationError("branch delete", "requires branch name: branch delete <name>")
	}

	name := ctx.GetArg(0)
	force := ctx.GetFlagBool("force")

	if name == "main" && !force {
		return orpheus.ValidationError("branch delete", "cannot delete main branch without --force").
			WithUserMessage("Deleting the main branch is a dangerous operation").
			WithContext("branch_name", name).
			WithContext("is_main_branch", true).
			WithContext("force_flag_provided", force).
			WithContext("required_flag", "--force").
			WithSeverity("critical")
	}

	if force {
		fmt.Printf("Force deleted branch '%s'\n", name)
	} else {
		fmt.Printf("Deleted branch '%s'\n", name)
	}

	return nil
}
