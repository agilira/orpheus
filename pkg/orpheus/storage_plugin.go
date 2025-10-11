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

	// pluginPaths contains search paths for plugin discovery
	pluginPaths []string

	// mutex protects concurrent access to the registry
	mutex sync.RWMutex

	// logger for plugin operations (optional, uses app logger if available)
	logger Logger

	// securityConfig controls plugin loading security checks
	securityConfig *PluginSecurityConfig
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
	// AllowUnsignedPlugins controls whether to load plugins without signatures
	AllowUnsignedPlugins bool `json:"allow_unsigned_plugins" yaml:"allow_unsigned_plugins"`

	// ValidateChecksums controls whether to verify plugin file integrity
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
			"./plugins",
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
		pluginPaths:    expandPluginPaths(configCopy.AllowedPaths),
		logger:         logger,
		securityConfig: configCopy,
	}
}

// LoadPlugin loads and validates a storage plugin from the specified path.
// Performs comprehensive security checks and caches the loaded plugin.
func (pm *PluginManager) LoadPlugin(ctx context.Context, pluginPath string) (*LoadedPlugin, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Validate plugin path security
	if err := pm.validatePluginPath(pluginPath); err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}

	// Check if already loaded
	pluginName := filepath.Base(pluginPath)
	if existing, exists := pm.registry[pluginName]; exists {
		pm.logDebug(ctx, "Plugin already loaded", "plugin", pluginName, "path", existing.Path)
		return existing, nil
	}

	// Validate file security
	if err := pm.validatePluginFile(pluginPath); err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}

	// Calculate file hash for integrity
	hash, err := pm.calculateFileHash(pluginPath)
	if err != nil {
		return nil, PluginLoadError(pluginPath, fmt.Errorf("failed to calculate file hash: %w", err))
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, PluginLoadError(pluginPath, fmt.Errorf("failed to open plugin: %w", err))
	}

	// Validate required symbols
	if err := pm.validatePluginSymbols(p); err != nil {
		return nil, PluginLoadError(pluginPath, err)
	}

	// Get the storage plugin constructor
	symbol, err := p.Lookup("NewStoragePlugin")
	if err != nil {
		return nil, PluginLoadError(pluginPath, fmt.Errorf("NewStoragePlugin symbol not found: %w", err))
	}

	// Validate the constructor signature
	constructor, ok := symbol.(func() StoragePlugin)
	if !ok {
		return nil, PluginLoadError(pluginPath, fmt.Errorf("NewStoragePlugin has invalid signature"))
	}

	// Create the plugin instance
	storagePlugin := constructor()
	if storagePlugin == nil {
		return nil, PluginLoadError(pluginPath, fmt.Errorf("NewStoragePlugin returned nil"))
	}

	// Create loaded plugin record
	loadedPlugin := &LoadedPlugin{
		Plugin:   storagePlugin,
		Path:     pluginPath,
		Hash:     hash,
		LoadTime: time.Now(),
		Metadata: map[string]interface{}{
			"name":        storagePlugin.Name(),
			"version":     storagePlugin.Version(),
			"description": storagePlugin.Description(),
		},
	}

	// Cache in registry
	pm.registry[storagePlugin.Name()] = loadedPlugin

	pm.logInfo(ctx, "Plugin loaded successfully",
		"plugin", storagePlugin.Name(),
		"version", storagePlugin.Version(),
		"path", pluginPath,
		"hash", hash[:12]+"...")

	return loadedPlugin, nil
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
	var plugins []string

	for _, searchPath := range pm.pluginPaths {
		found, err := pm.discoverPluginsInPath(ctx, searchPath)
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
	if config.PluginPath != "" {
		return pm.LoadPlugin(ctx, config.PluginPath)
	}

	// Try to find plugin by provider name
	discovered, err := pm.DiscoverPlugins(ctx)
	if err != nil {
		return nil, ConfigValidationError(config.Provider, fmt.Errorf("plugin discovery failed: %w", err))
	}

	// Look for plugin matching the provider name
	for _, pluginPath := range discovered {
		if strings.Contains(filepath.Base(pluginPath), config.Provider) {
			return pm.LoadPlugin(ctx, pluginPath)
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

	pm.logInfo(ctx, "Plugin unloaded", "plugin", name, "path", plugin.Path)
	return nil
}

// Private helper methods

func (pm *PluginManager) validatePluginPath(pluginPath string) error {
	// Check if path is absolute
	if !filepath.IsAbs(pluginPath) {
		return fmt.Errorf("plugin path must be absolute: %s", pluginPath)
	}

	// Validate against allowed paths if configured
	if len(pm.pluginPaths) > 0 {
		allowed := false
		for _, allowedPath := range pm.pluginPaths {
			if strings.HasPrefix(pluginPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("plugin path not in allowed paths: %s", pluginPath)
		}
	}

	return nil
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

	data, err := os.ReadFile(pluginPath)
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

	if err != nil && !os.IsNotExist(err) {
		// Check if it's one of our security-related errors
		if strings.Contains(err.Error(), "timeout exceeded") ||
			strings.Contains(err.Error(), "too many files") ||
			err == context.DeadlineExceeded ||
			err == context.Canceled {
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
		// Handle ~ expansion
		if strings.HasPrefix(path, "~/") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(homeDir, path[2:])
			}
		}

		// Convert to absolute path
		if absPath, err := filepath.Abs(path); err == nil {
			expanded = append(expanded, absPath)
		} else {
			expanded = append(expanded, path)
		}
	}

	return expanded
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
