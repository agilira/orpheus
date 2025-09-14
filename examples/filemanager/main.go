// main.go: File Manager example using Orpheus framework
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func main() {
	// Create the main application
	app := orpheus.New("filemanager").
		SetDescription("A simple file manager CLI demonstrating Orpheus framework").
		SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output").
		AddGlobalFlag("config", "c", "", "Configuration file path")

	// Add commands to demonstrate different features
	setupListCommand(app)
	setupSearchCommand(app)
	setupInfoCommand(app)
	setupTreeCommand(app)

	// Add built-in completion support
	app.AddCompletionCommand()

	// Run the application with enhanced error handling
	if err := app.Run(os.Args[1:]); err != nil {
		if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
			// Display user-friendly message
			fmt.Fprintf(os.Stderr, "Error: %s\n", orpheusErr.UserMessage())

			// Show technical details if verbose
			if os.Getenv("DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "Technical details: %s\n", orpheusErr.Error())
				fmt.Fprintf(os.Stderr, "Error code: %s\n", orpheusErr.ErrorCode())
			}

			// Indicate if retryable
			if orpheusErr.IsRetryable() {
				fmt.Fprintf(os.Stderr, "This operation can be retried.\n")
			}

			os.Exit(orpheusErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

// setupListCommand demonstrates basic command with flags
func setupListCommand(app *orpheus.App) {
	cmd := orpheus.NewCommand("list", "List files and directories").
		SetHandler(listHandler).
		AddFlag("path", "p", ".", "Path to list").
		AddBoolFlag("all", "a", false, "Show hidden files").
		AddBoolFlag("long", "l", false, "Use long listing format").
		AddIntFlag("limit", "n", 0, "Limit number of results (0 = no limit)")

	app.AddCommand(cmd)
}

// setupSearchCommand demonstrates string slice flags
func setupSearchCommand(app *orpheus.App) {
	cmd := orpheus.NewCommand("search", "Search for files by pattern").
		SetHandler(searchHandler).
		AddFlag("pattern", "p", "*", "Search pattern (glob)").
		AddFlag("dir", "d", ".", "Directory to search in").
		AddStringSliceFlag("ext", "e", []string{}, "File extensions to include").
		AddBoolFlag("recursive", "r", true, "Search recursively").
		SetLongDescription(`Search for files matching the specified pattern.
Supports glob patterns like *.go, test*, etc.
Use --ext to filter by file extensions.`).
		AddExample("filemanager search --pattern '*.go' --dir ./src").
		AddExample("filemanager search -p 'test*' --ext go,txt")

	app.AddCommand(cmd)
}

// setupInfoCommand demonstrates argument handling
func setupInfoCommand(app *orpheus.App) {
	cmd := orpheus.NewCommand("info", "Show detailed information about a file or directory").
		SetHandler(infoHandler).
		AddBoolFlag("size", "s", true, "Show file size").
		AddBoolFlag("permissions", "p", true, "Show permissions").
		AddBoolFlag("timestamps", "t", false, "Show detailed timestamps")

	app.AddCommand(cmd)
}

// setupTreeCommand demonstrates complex nested functionality
func setupTreeCommand(app *orpheus.App) {
	cmd := orpheus.NewCommand("tree", "Display directory tree structure").
		SetHandler(treeHandler).
		AddFlag("root", "r", ".", "Root directory for tree").
		AddIntFlag("depth", "d", 3, "Maximum depth to display").
		AddBoolFlag("dirs-only", "D", false, "Show directories only")

	app.AddCommand(cmd)
}

// Command handlers

func listHandler(ctx *orpheus.Context) error {
	listParams := extractListParams(ctx)

	if listParams.verbose {
		fmt.Printf("Listing directory: %s\n", listParams.path)
	}

	entries, err := readDirectory(listParams.path)
	if err != nil {
		return err
	}

	count := displayEntries(entries, listParams)

	if listParams.verbose {
		fmt.Printf("Listed %d items\n", count)
	}

	return nil
}

// listParams holds listing configuration
type listParams struct {
	path       string
	showAll    bool
	longFormat bool
	limit      int
	verbose    bool
}

// extractListParams extracts listing parameters from context
func extractListParams(ctx *orpheus.Context) listParams {
	return listParams{
		path:       ctx.GetFlagString("path"),
		showAll:    ctx.GetFlagBool("all"),
		longFormat: ctx.GetFlagBool("long"),
		limit:      ctx.GetFlagInt("limit"),
		verbose:    ctx.GetGlobalFlagBool("verbose"),
	}
}

// readDirectory reads directory entries with error handling
func readDirectory(path string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, orpheus.ExecutionError("list", fmt.Sprintf("cannot read directory %s: %v", path, err)).
			WithUserMessage("Unable to read the specified directory").
			WithContext("path", path).
			WithContext("error_type", "permission_or_not_found").
			AsRetryable()
	}
	return entries, nil
}

// displayEntries formats and displays directory entries
func displayEntries(entries []fs.DirEntry, params listParams) int {
	count := 0

	for _, entry := range entries {
		if shouldSkipEntry(entry, params, count) {
			continue
		}

		displayEntry(entry, params.longFormat)
		count++
	}

	return count
}

// shouldSkipEntry determines if an entry should be skipped
func shouldSkipEntry(entry fs.DirEntry, params listParams, count int) bool {
	// Skip hidden files unless --all is specified
	if !params.showAll && strings.HasPrefix(entry.Name(), ".") {
		return true
	}

	// Apply limit if specified
	if params.limit > 0 && count >= params.limit {
		return true
	}

	return false
}

// displayEntry displays a single directory entry
func displayEntry(entry fs.DirEntry, longFormat bool) {
	if longFormat {
		displayLongFormat(entry)
	} else {
		displayShortFormat(entry)
	}
}

// displayLongFormat displays entry in long format
func displayLongFormat(entry fs.DirEntry) {
	info, err := entry.Info()
	if err != nil {
		return
	}
	fmt.Printf("%s %8d %s %s\n",
		info.Mode(),
		info.Size(),
		info.ModTime().Format("Jan 02 15:04"),
		entry.Name())
}

// displayShortFormat displays entry in short format
func displayShortFormat(entry fs.DirEntry) {
	if entry.IsDir() {
		fmt.Printf("%s/\n", entry.Name())
	} else {
		fmt.Printf("%s\n", entry.Name())
	}
}

func searchHandler(ctx *orpheus.Context) error {
	searchParams := extractSearchParams(ctx)

	if searchParams.verbose {
		logSearchStart(searchParams)
	}

	matches, err := performSearch(searchParams)
	if err != nil {
		return err
	}

	displayResults(matches, searchParams.verbose)
	return nil
}

// searchParams holds search configuration
type searchParams struct {
	pattern    string
	dir        string
	extensions []string
	recursive  bool
	verbose    bool
}

// extractSearchParams extracts search parameters from context
func extractSearchParams(ctx *orpheus.Context) searchParams {
	return searchParams{
		pattern:    ctx.GetFlagString("pattern"),
		dir:        ctx.GetFlagString("dir"),
		extensions: ctx.GetFlagStringSlice("ext"),
		recursive:  ctx.GetFlagBool("recursive"),
		verbose:    ctx.GetGlobalFlagBool("verbose"),
	}
}

// logSearchStart logs the start of search operation
func logSearchStart(params searchParams) {
	fmt.Printf("Searching for pattern '%s' in %s\n", params.pattern, params.dir)
	if len(params.extensions) > 0 {
		fmt.Printf("Filtering extensions: %s\n", strings.Join(params.extensions, ", "))
	}
}

// performSearch executes the file search
func performSearch(params searchParams) ([]string, error) {
	var matches []string

	walkFunc := createWalkFunc(&matches, params)
	err := filepath.WalkDir(params.dir, walkFunc)

	if err != nil {
		return nil, orpheus.ExecutionError("search", fmt.Sprintf("search failed: %v", err)).
			WithUserMessage("Unable to search in the specified directory").
			WithContext("directory", params.dir).
			WithContext("pattern", params.pattern).
			AsRetryable()
	}

	return matches, nil
}

// createWalkFunc creates the walk function for file traversal
func createWalkFunc(matches *[]string, params searchParams) func(string, fs.DirEntry, error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if d.IsDir() {
			return handleDirectory(path, params.dir, params.recursive)
		}

		return handleFile(path, matches, params)
	}
}

// handleDirectory processes directory entries during walk
func handleDirectory(path, rootDir string, recursive bool) error {
	// If not recursive and this is not the root directory, skip
	if !recursive && path != rootDir {
		return filepath.SkipDir
	}
	return nil // Continue into directory
}

// handleFile processes file entries during walk
func handleFile(path string, matches *[]string, params searchParams) error {
	if !matchesExtension(path, params.extensions) {
		return nil
	}

	if matchesPattern(path, params.pattern) {
		*matches = append(*matches, path)
	}

	return nil
}

// matchesExtension checks if file matches extension filter
func matchesExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}

	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, allowedExt := range extensions {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

// matchesPattern checks if file matches the search pattern
func matchesPattern(path, pattern string) bool {
	matched, err := filepath.Match(pattern, filepath.Base(path))
	return err == nil && matched
}

// displayResults shows search results to user
func displayResults(matches []string, verbose bool) {
	for _, match := range matches {
		fmt.Println(match)
	}

	if verbose {
		fmt.Printf("Found %d matches\n", len(matches))
	}
}

func infoHandler(ctx *orpheus.Context) error {
	if ctx.ArgCount() == 0 {
		return orpheus.ValidationError("info", "missing file path argument").
			WithUserMessage("Please specify a file or directory path").
			WithContext("usage", "filemanager info <path>")
	}

	path := ctx.GetArg(0)
	showSize := ctx.GetFlagBool("size")
	showPerms := ctx.GetFlagBool("permissions")
	showTimestamps := ctx.GetFlagBool("timestamps")

	info, err := os.Stat(path)
	if err != nil {
		return orpheus.NotFoundError("info", fmt.Sprintf("cannot access %s: %v", path, err))
	}

	fmt.Printf("File: %s\n", path)

	if info.IsDir() {
		fmt.Println("Type: Directory")
	} else {
		fmt.Println("Type: Regular file")
	}

	if showSize {
		fmt.Printf("Size: %d bytes\n", info.Size())
	}

	if showPerms {
		fmt.Printf("Permissions: %s\n", info.Mode())
	}

	if showTimestamps {
		fmt.Printf("Modified: %s\n", info.ModTime().Format(time.RFC3339))
	} else {
		fmt.Printf("Modified: %s\n", info.ModTime().Format("Jan 02, 2006 15:04:05"))
	}

	return nil
}

func treeHandler(ctx *orpheus.Context) error {
	root := ctx.GetFlagString("root")
	maxDepth := ctx.GetFlagInt("depth")
	dirsOnly := ctx.GetFlagBool("dirs-only")
	verbose := ctx.GetGlobalFlagBool("verbose")

	if verbose {
		fmt.Printf("Building tree from %s (max depth: %d)\n", root, maxDepth)
	}

	fmt.Printf("%s\n", root)
	return printTree(root, "", 0, maxDepth, dirsOnly)
}

func printTree(path string, prefix string, depth int, maxDepth int, dirsOnly bool) error {
	if depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip files if dirs-only is enabled
		if dirsOnly && !entry.IsDir() {
			continue
		}

		isLast := i == len(entries)-1
		var connector, newPrefix string

		if isLast {
			connector = "└── "
			newPrefix = prefix + "    "
		} else {
			connector = "├── "
			newPrefix = prefix + "│   "
		}

		if entry.IsDir() {
			fmt.Printf("%s%s%s/\n", prefix, connector, entry.Name())
			_ = printTree(filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth, dirsOnly)
		} else {
			fmt.Printf("%s%s%s\n", prefix, connector, entry.Name())
		}
	}

	return nil
}
