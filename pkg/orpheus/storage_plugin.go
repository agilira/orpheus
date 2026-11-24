// Plugin Loading System for Orpheus Storage
//
// This package implements secure dynamic loading of storage provider plugins
// using Go's plugin system. Provides discovery, validation, and lifecycle
// management for storage backend plugins with comprehensive security checks.
//
// Features:
// - Dynamic .so plugin loading with security validation
// - Auto-discovery from standard plugin locations
// - Plugin lifecycle management (load, validate, unload)
// - Thread-safe plugin registry with caching
// - Comprehensive error handling and logging
// - Security hardening against malicious plugins
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"
)

// PluginManager manages the lifecycle of storage plugins with security validation
// and caching. Thread-safe and designed for high-performance production use.
type PluginManager struct {
	// registry holds loaded plugins by name for fast lookup
	registry map[string]*LoadedPlugin

	// byPath indexes the same plugins by resolved file path.
	// WHY: the file name says nothing about Name(), so only the resolved
	// path tells that a file was already opened and its constructor run.
	byPath map[string]*LoadedPlugin

	// pluginPaths contains search paths for plugin discovery
	pluginPaths []string

	// mutex protects concurrent access to the registry
	mutex sync.RWMutex

	// logger for plugin operations (optional, uses app logger if available)
	logger Logger

	// securityConfig controls plugin loading security checks
	securityConfig *PluginSecurityConfig

	// restrictPaths is true when AllowedPaths was configured.
	// WHY: entries can be dropped on expansion ("~" without a home); an
	// allow-list emptied that way must allow nothing, not everything.
	restrictPaths bool
}

// LoadedPlugin represents a successfully loaded and validated storage plugin
type LoadedPlugin struct {
	// Plugin is the loaded StoragePlugin interface
	Plugin StoragePlugin

	// Path is the filesystem path where the plugin was loaded from
	Path string

	// Hash is the SHA256 hash of the plugin file for integrity verification
	Hash string

	// LoadTime is when the plugin was successfully loaded
	LoadTime time.Time

	// Metadata contains additional plugin information
	Metadata map[string]interface{}
}

// PluginSecurityConfig controls security validation during plugin loading
type PluginSecurityConfig struct {
	// AllowUnsignedPlugins controls whether to load plugins without signatures.
	// Signature verification is not implemented, so with false (the default)
	// every plugin is refused; set it to true to load plugins at all.
	AllowUnsignedPlugins bool `json:"allow_unsigned_plugins" yaml:"allow_unsigned_plugins"`

	// ValidateChecksums controls whether the SHA-256 of the plugin file is
	// computed and recorded in LoadedPlugin.Hash. There is no expected
	// checksum to compare it with, so it records integrity, it does not enforce it.
	ValidateChecksums bool `json:"validate_checksums" yaml:"validate_checksums"`

	// MaxPluginSize is the maximum allowed plugin file size in bytes
	MaxPluginSize int64 `json:"max_plugin_size_bytes" yaml:"max_plugin_size_bytes"`

	// AllowedPaths restricts plugin loading to specific directories
	AllowedPaths []string `json:"allowed_paths,omitempty" yaml:"allowed_paths,omitempty"`

	// RequiredSymbols are the symbols that must be present in the plugin
	RequiredSymbols []string `json:"required_symbols,omitempty" yaml:"required_symbols,omitempty"`
}

// DefaultPluginSecurityConfig returns a secure default configuration
func DefaultPluginSecurityConfig() *PluginSecurityConfig {
	return &PluginSecurityConfig{
		AllowUnsignedPlugins: false,
		ValidateChecksums:    true,
		MaxPluginSize:        50 << 20, // 50MB
		AllowedPaths: []string{
			"/usr/local/lib/orpheus/plugins",
			"/opt/orpheus/plugins",
			// No path relative to the working directory: running a CLI
			// inside a cloned repository must not load plugins it ships.
			"~/.orpheus/plugins",
		},
		RequiredSymbols: []string{
			"NewStoragePlugin",
		},
	}
}

// NewPluginManager creates a new plugin manager with the specified configuration
func NewPluginManager(logger Logger, securityConfig *PluginSecurityConfig) *PluginManager {
	if securityConfig == nil {
		securityConfig = DefaultPluginSecurityConfig()
	}

	// Create defensive copy of security config to prevent external modifications
	configCopy := &PluginSecurityConfig{
		AllowUnsignedPlugins: securityConfig.AllowUnsignedPlugins,
		ValidateChecksums:    securityConfig.ValidateChecksums,
		MaxPluginSize:        securityConfig.MaxPluginSize,
		AllowedPaths:         make([]string, len(securityConfig.AllowedPaths)),
		RequiredSymbols:      make([]string, len(securityConfig.RequiredSymbols)),
	}
	copy(configCopy.AllowedPaths, securityConfig.AllowedPaths)
	copy(configCopy.RequiredSymbols, securityConfig.RequiredSymbols)

	return &PluginManager{
		registry:       make(map[string]*LoadedPlugin),
		byPath:         make(map[string]*LoadedPlugin),
		pluginPaths:    expandPluginPaths(configCopy.AllowedPaths),
		logger:         logger,
		securityConfig: configCopy,
		restrictPaths:  len(configCopy.AllowedPaths) > 0,
	}
}

// LoadPlugin loads and validates a storage plugin from the specified path.
// Performs comprehensive security checks and caches the loaded plugin.
// A path that resolves to a plugin already loaded returns the cached record.
func (pm *PluginManager) LoadPlugin(ctx context.Context, pluginPath string) (*LoadedPlugin, error) {
	return pm.loadPlugin(ctx, pluginPath, pm.securityConfig.AllowUnsignedPlugins)
}

// errSignatureUnavailable explains why every plugin is refused by default.
var errSignatureUnavailable = fmt.Errorf("plugin signature verification is not available; " +
	"set AllowUnsignedPlugins to true to load unsigned plugins")

func (pm *PluginManager) loadPlugin(ctx context.Context, pluginPath string, allowUnsigned bool) (*LoadedPlugin, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if err := pm.validatePluginPath(pluginPath); err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}
	// From here on only the resolved path is used, so the file hashed and
	// opened is the one that was checked.
	realPath, err := pm.resolvePluginFile(pluginPath)
	if err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}
	if existing, ok := pm.byPath[realPath]; ok {
		pm.logDebug(ctx, "Plugin already loaded", "plugin", existing.Plugin.Name(), "path", existing.Path)
		return existing, nil
	}
	// WHY: no signature check exists, so "unsigned not allowed" must mean
	// nothing is loaded rather than everything.
	if !allowUnsigned {
		return nil, PluginLoadError(pluginPath, errSignatureUnavailable)
	}
	hash, err := pm.calculateFileHash(realPath)
	if err != nil {
		return nil, PluginLoadError(pluginPath, fmt.Errorf("failed to calculate file hash: %w", err))
	}
	storagePlugin, err := pm.openPlugin(realPath)
	if err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}
	loaded, err := pm.register(storagePlugin, pluginPath, realPath, hash)
	if err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}
	pm.logInfo(ctx, "Plugin loaded successfully",
		"plugin", storagePlugin.Name(),
		"version", storagePlugin.Version(),
		"path", pluginPath,
		"hash", shortHash(hash))
	return loaded, nil
}

// openPlugin opens the file, checks its symbols and runs its constructor.
func (pm *PluginManager) openPlugin(realPath string) (StoragePlugin, error) {
	p, err := plugin.Open(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}
	if err := pm.validatePluginSymbols(p); err != nil {
		return nil, err
	}
	symbol, err := p.Lookup("NewStoragePlugin")
	if err != nil {
		return nil, fmt.Errorf("NewStoragePlugin symbol not found: %w", err)
	}
	constructor, ok := symbol.(func() StoragePlugin)
	if !ok {
		return nil, fmt.Errorf("NewStoragePlugin has invalid signature")
	}
	storagePlugin := constructor()
	if storagePlugin == nil {
		return nil, fmt.Errorf("NewStoragePlugin returned nil")
	}
	return storagePlugin, nil
}

// register records a new plugin under its name and resolved path. A second
// file claiming a loaded name is refused: replacing the entry would leave
// the first file's path pointing at a record no longer in the registry.
func (pm *PluginManager) register(sp StoragePlugin, pluginPath, realPath, hash string) (*LoadedPlugin, error) {
	name := sp.Name()
	if existing, ok := pm.registry[name]; ok {
		return nil, fmt.Errorf("plugin name %q already loaded from %s", name, existing.Path)
	}
	loaded := &LoadedPlugin{
		Plugin:   sp,
		Path:     pluginPath,
		Hash:     hash,
		LoadTime: time.Now(),
		Metadata: map[string]interface{}{
			"name":        name,
			"version":     sp.Version(),
			"description": sp.Description(),
		},
	}
	pm.registry[name] = loaded
	pm.byPath[realPath] = loaded
	return loaded, nil
}

// shortHash abbreviates a hash for logs; it is empty when checksums are off.
func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12] + "..."
}

// GetPlugin retrieves a loaded plugin by name
func (pm *PluginManager) GetPlugin(name string) (*LoadedPlugin, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.registry[name]
	if !exists {
		return nil, NewError(ErrCodePluginError, "storage", fmt.Sprintf("plugin '%s' not found in registry", name)).
			WithContext("operation", "storage.GetPlugin").
			WithContext("plugin_name", name).
			WithSeverity("error")
	}

	return plugin, nil
}

// DiscoverPlugins searches for plugins in configured paths and returns found plugin files
func (pm *PluginManager) DiscoverPlugins(ctx context.Context) ([]string, error) {
	// WHY: with no search path the loop below never runs, and a cancelled
	// search would come back as "nothing found".
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var plugins []string

	for _, searchPath := range pm.pluginPaths {
		found, err := pm.discoverPluginsInPath(ctx, searchPath)
		// WHY: a cancelled search finds nothing, which callers would
		// otherwise report as "no plugin found".
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if err != nil {
			pm.logWarn(ctx, "Failed to discover plugins in path", "path", searchPath, "error", err)
			continue
		}
		plugins = append(plugins, found...)
	}

	pm.logInfo(ctx, "Plugin discovery completed", "found_count", len(plugins), "search_paths", len(pm.pluginPaths))
	return plugins, nil
}

// LoadPluginsFromConfig loads storage plugins based on storage configuration
func (pm *PluginManager) LoadPluginsFromConfig(ctx context.Context, config *StorageConfig) (*LoadedPlugin, error) {
	// If specific plugin path is provided, use it directly
	allowUnsigned := pm.securityConfig.AllowUnsignedPlugins || config.AllowUnsignedPlugins
	if config.PluginPath != "" {
		return pm.loadPlugin(ctx, config.PluginPath, allowUnsigned)
	}

	// An empty name would match any plugin found on the search paths.
	if config.Provider == "" {
		return nil, ConfigValidationError(config.Provider, fmt.Errorf("provider name is empty"))
	}

	// Try to find plugin by provider name
	discovered, err := pm.DiscoverPlugins(ctx)
	if err != nil {
		return nil, ConfigValidationError(config.Provider, fmt.Errorf("plugin discovery failed: %w", err))
	}

	// Look for plugin matching the provider name
	for _, pluginPath := range discovered {
		// Exact match only: a substring would let "mem" load memory-evil.so.
		if strings.TrimSuffix(filepath.Base(pluginPath), ".so") == config.Provider {
			return pm.loadPlugin(ctx, pluginPath, allowUnsigned)
		}
	}

	return nil, NewError(ErrCodePluginError, "storage", fmt.Sprintf("no plugin found for provider '%s'", config.Provider)).
		WithContext("operation", "storage.LoadPluginsFromConfig").
		WithContext("provider", config.Provider).
		WithSeverity("error").
		WithUserMessage(fmt.Sprintf("Storage provider '%s' is not available", config.Provider))
}

// ListLoadedPlugins returns information about all loaded plugins
func (pm *PluginManager) ListLoadedPlugins() map[string]*LoadedPlugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]*LoadedPlugin, len(pm.registry))
	for name, plugin := range pm.registry {
		result[name] = plugin
	}

	return result
}

// UnloadPlugin removes a plugin from the registry (Go doesn't support true plugin unloading)
func (pm *PluginManager) UnloadPlugin(ctx context.Context, name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	plugin, exists := pm.registry[name]
	if !exists {
		return NewError(ErrCodePluginError, "storage", fmt.Sprintf("plugin '%s' not loaded", name)).
			WithContext("operation", "storage.UnloadPlugin").
			WithContext("plugin_name", name).
			WithSeverity("warning")
	}

	delete(pm.registry, name)
	for path, loaded := range pm.byPath {
		if loaded == plugin {
			delete(pm.byPath, path)
		}
	}

	pm.logInfo(ctx, "Plugin unloaded", "plugin", name, "path", plugin.Path)
	return nil
}

// Private helper methods

func (pm *PluginManager) validatePluginPath(pluginPath string) error {
	// Check if path is absolute
	if !filepath.IsAbs(pluginPath) {
		return fmt.Errorf("plugin path must be absolute: %s", pluginPath)
	}

	// The OS resolves ".." after following symlinks, Clean does not, so a
	// path that is not already clean could be judged on a different file.
	if filepath.Clean(pluginPath) != pluginPath {
		return fmt.Errorf("plugin path is not canonical: %s", pluginPath)
	}

	// Validate against allowed paths if configured
	if pm.restrictPaths {
		for _, allowedPath := range pm.pluginPaths {
			if isWithinDir(allowedPath, pluginPath) {
				return nil
			}
		}
		return fmt.Errorf("plugin path not in allowed paths: %s", pluginPath)
	}

	return nil
}

// isWithinDir reports whether path lies strictly below dir. A string prefix
// is not enough: "/opt/plugins-evil" starts with "/opt/plugins".
func isWithinDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolvePluginFile validates the file and returns its path with every
// symlink resolved. The resolved path must still lie below an allowed
// directory (itself resolved), otherwise a link placed in an allowed
// directory would load a file from anywhere.
func (pm *PluginManager) resolvePluginFile(pluginPath string) (string, error) {
	if err := pm.validatePluginFile(pluginPath); err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(pluginPath)
	if err != nil {
		return "", fmt.Errorf("plugin path cannot be resolved: %w", err)
	}
	if !pm.restrictPaths {
		return realPath, nil
	}
	for _, dir := range pm.pluginPaths {
		realDir, err := filepath.EvalSymlinks(dir)
		if err == nil && isWithinDir(realDir, realPath) {
			return realPath, nil
		}
	}
	return "", fmt.Errorf("plugin resolves outside allowed paths: %s -> %s", pluginPath, realPath)
}

func (pm *PluginManager) validatePluginFile(pluginPath string) error {
	stat, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin file not accessible: %w", err)
	}

	if !stat.Mode().IsRegular() {
		return fmt.Errorf("plugin path is not a regular file: %s", pluginPath)
	}

	if pm.securityConfig.MaxPluginSize > 0 && stat.Size() > pm.securityConfig.MaxPluginSize {
		return fmt.Errorf("plugin file too large: %d bytes (max: %d)", stat.Size(), pm.securityConfig.MaxPluginSize)
	}

	return nil
}

func (pm *PluginManager) validatePluginSymbols(p *plugin.Plugin) error {
	for _, symbolName := range pm.securityConfig.RequiredSymbols {
		_, err := p.Lookup(symbolName)
		if err != nil {
			return fmt.Errorf("required symbol '%s' not found: %w", symbolName, err)
		}
	}
	return nil
}

func (pm *PluginManager) calculateFileHash(pluginPath string) (string, error) {
	if !pm.securityConfig.ValidateChecksums {
		return "", nil
	}

	data, err := os.ReadFile(pluginPath) // #nosec G304 -- pluginPath is validated by validatePluginPath() and validatePluginFile() before reaching this point
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

func (pm *PluginManager) discoverPluginsInPath(ctx context.Context, searchPath string) ([]string, error) {
	var plugins []string

	// Security limits to prevent DoS attacks
	const (
		maxDepth    = 10              // Maximum directory depth
		maxFiles    = 10000           // Maximum files to examine
		maxDuration = 2 * time.Second // Maximum time allowed for discovery
	)

	startTime := time.Now()
	filesExamined := 0
	visitedDirs := make(map[string]bool) // Prevent infinite loops with symlinks

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check timeout
		if time.Since(startTime) > maxDuration {
			return fmt.Errorf("plugin discovery timeout exceeded for path: %s", searchPath)
		}

		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Count files/directories examined
		filesExamined++
		if filesExamined > maxFiles {
			return fmt.Errorf("too many files examined during plugin discovery in path: %s", searchPath)
		}

		if info.IsDir() {
			// Check depth limit
			relPath, err := filepath.Rel(searchPath, path)
			if err == nil {
				depth := strings.Count(relPath, string(filepath.Separator))
				if depth > maxDepth {
					return filepath.SkipDir
				}
			}

			// Prevent infinite loops with symlinks
			if info.Mode()&os.ModeSymlink != 0 {
				realPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					return nil // Skip broken symlinks
				}
				if visitedDirs[realPath] {
					return filepath.SkipDir // Skip already visited directories
				}
				visitedDirs[realPath] = true
			}

			return nil
		}

		// Look for .so files
		if strings.HasSuffix(path, ".so") {
			plugins = append(plugins, path)
		}

		return nil
	})

	// WHY: the caller's cancellation is not a security limit; it is
	// returned as is so that callers can tell it from "nothing found".
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil && !os.IsNotExist(err) {
		// Check if it's one of our security-related errors
		if strings.Contains(err.Error(), "timeout exceeded") ||
			strings.Contains(err.Error(), "too many files") {
			pm.logWarn(ctx, "Plugin discovery terminated due to security limits",
				"path", searchPath,
				"error", err.Error(),
				"files_examined", filesExamined,
				"duration", time.Since(startTime))
			return plugins, nil // Return what we found so far
		}
		return nil, err
	}

	pm.logDebug(ctx, "Plugin discovery completed",
		"path", searchPath,
		"plugins_found", len(plugins),
		"files_examined", filesExamined,
		"duration", time.Since(startTime))

	return plugins, nil
}

func expandPluginPaths(paths []string) []string {
	var expanded []string
	for _, path := range paths {
		if p, ok := expandPluginPath(path); ok {
			expanded = append(expanded, p)
		}
	}
	return expanded
}

// expandPluginPath expands "~/" and makes the path absolute. A "~" entry is
// dropped when the home directory is unknown.
// WHY: left as is, "~/.orpheus/plugins" is relative, and Abs would turn it
// into a directory under the working directory, allowed implicitly.
func expandPluginPath(path string) (string, bool) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", false
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	if absPath, err := filepath.Abs(path); err == nil {
		return absPath, true
	}
	return path, true
}

func (pm *PluginManager) logDebug(ctx context.Context, message string, args ...interface{}) {
	if pm.logger != nil {
		// Convert args to Field slice for the Logger interface
		fields := make([]Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				fields = append(fields, Field{Key: key, Value: args[i+1]})
			}
		}
		pm.logger.Debug(ctx, message, fields...)
	}
}

func (pm *PluginManager) logInfo(ctx context.Context, message string, args ...interface{}) {
	if pm.logger != nil {
		// Convert args to Field slice for the Logger interface
		fields := make([]Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				fields = append(fields, Field{Key: key, Value: args[i+1]})
			}
		}
		pm.logger.Info(ctx, message, fields...)
	}
}

func (pm *PluginManager) logWarn(ctx context.Context, message string, args ...interface{}) {
	if pm.logger != nil {
		// Convert args to Field slice for the Logger interface
		fields := make([]Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				fields = append(fields, Field{Key: key, Value: args[i+1]})
			}
		}
		pm.logger.Warn(ctx, message, fields...)
	}
}
