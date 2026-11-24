// validation.go: Secure CLI input validation for Orpheus framework
//
// This module provides comprehensive input validation specifically designed for CLI applications,
// integrating security controls to prevent common CLI attack vectors while maintaining
// high performance for responsive user experience.
//
// SECURITY INTEGRATION:
// - Automatic path security validation for file/directory flags
// - Environment variable sanitization and validation
// - Command injection prevention in CLI arguments
// - File permission checking before operations
// - Input sanitization with performance optimization
//
// PERFORMANCE PHILOSOPHY:
// - Zero-allocation validation for common cases
// - Lazy evaluation of expensive checks
// - Caching of validation results
// - Minimal overhead for legitimate inputs
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ValidationConfig controls the behavior of CLI input validation
type ValidationConfig struct {
	// Security controls
	SecurityConfig SecurityConfig

	// Path validation options
	ValidatePathArgs bool // Validate path arguments for security
	ValidateFileArgs bool // Check file existence and permissions
	NormalizePaths   bool // Normalize paths before validation

	// Environment variable validation
	ValidateEnvVars bool // Validate environment variables used in defaults
	// TrustedEnvPrefixes lists trusted environment variables: an entry is an
	// exact name ("HOME"), or a prefix when it ends with "_" ("MYAPP_").
	TrustedEnvPrefixes []string

	// Input sanitization
	SanitizeInputs   bool // Remove/escape dangerous characters
	MaxArgLength     int  // Maximum length for individual arguments
	MaxTotalArgsSize int  // Maximum total size of all arguments

	// Performance options
	EnableCaching  bool // Cache validation results
	CacheSize      int  // Maximum cache entries
	LazyValidation bool // Validate only when needed
}

// DefaultValidationConfig returns a secure default validation configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		SecurityConfig:     DefaultSecurityConfig(),
		ValidatePathArgs:   true,
		ValidateFileArgs:   true,
		NormalizePaths:     true,
		ValidateEnvVars:    true,
		TrustedEnvPrefixes: []string{"ORPHEUS_", "HOME", "PATH", "USER"},
		SanitizeInputs:     true,
		MaxArgLength:       4096,
		MaxTotalArgsSize:   65536, // 64KB total
		EnableCaching:      true,
		CacheSize:          1000,
		LazyValidation:     false,
	}
}

// ValidatedInput represents the result of input validation
type ValidatedInput struct {
	OriginalValue     string          // Original input value
	SanitizedValue    string          // Sanitized/normalized value to use
	IsValid           bool            // Whether input passes validation
	ValidationErrors  []string        // List of validation errors
	SecurityWarnings  []string        // List of security warnings
	FilePermissions   *FilePermission // File permission info (if applicable)
	RecommendedAction string          // Recommended action for the application
}

// InputValidator provides secure validation for CLI inputs with caching
type InputValidator struct {
	config  ValidationConfig
	cache   map[string]*ValidatedInput
	cacheMu sync.RWMutex
}

// NewInputValidator creates a new CLI input validator with the specified configuration
func NewInputValidator(config ValidationConfig) *InputValidator {
	return &InputValidator{
		config: config,
		cache:  make(map[string]*ValidatedInput),
	}
}

// ValidatePathFlag validates a path flag value with comprehensive security checks
//
// This is the primary method for validating file/directory paths provided via CLI flags.
// It integrates path security validation, file permission analysis, and normalization.
//
// Performance (BenchmarkValidatePathFlag): ~220ns and 2 allocations for a
// cached result; a new one includes file system access. Caching is what makes
// this cheap -- it is on by default.
func (v *InputValidator) ValidatePathFlag(flagName, value string) *ValidatedInput {
	key, cached := v.lookupCache(cacheKindPath, flagName, value)
	if cached != nil {
		return cached
	}

	result := &ValidatedInput{
		OriginalValue:     value,
		SanitizedValue:    value,
		IsValid:           true,
		ValidationErrors:  make([]string, 0),
		SecurityWarnings:  make([]string, 0),
		RecommendedAction: "proceed",
	}

	// Step 1: Basic input validation
	if !v.validateBasicInput(value, result) {
		return v.cacheAndReturn(key, result)
	}

	// Step 2: Path normalization
	if v.config.NormalizePaths {
		result.SanitizedValue = filepath.Clean(value)
	}

	// Step 3: Security path validation
	if v.config.ValidatePathArgs {
		pathResult := ValidateSecurePath(result.SanitizedValue, v.config.SecurityConfig)
		if !pathResult.IsValid {
			result.IsValid = false
			result.ValidationErrors = append(result.ValidationErrors, pathResult.Errors...)
			result.RecommendedAction = "reject_dangerous_path"
			return v.cacheAndReturn(key, result)
		}

		// Add security warnings if any
		result.SecurityWarnings = append(result.SecurityWarnings, pathResult.Warnings...)
		result.SanitizedValue = pathResult.NormalizedPath
	}

	// Step 4: File permission analysis (if file exists)
	if v.config.ValidateFileArgs {
		v.analyzeFilePermissions(result)
	}

	return v.cacheAndReturn(key, result)
}

// ValidateStringFlag validates a general string flag with security checks
//
// Used for non-path string flags that may still pose security risks through
// command injection, environment variable injection, or other attack vectors.
//
// Performance (BenchmarkValidateStringFlag): ~500ns and 2 allocations with
// sanitization enabled, which is the default.
func (v *InputValidator) ValidateStringFlag(flagName, value string) *ValidatedInput {
	key, cached := v.lookupCache(cacheKindString, flagName, value)
	if cached != nil {
		return cached
	}

	result := &ValidatedInput{
		OriginalValue:     value,
		SanitizedValue:    value,
		IsValid:           true,
		ValidationErrors:  make([]string, 0),
		SecurityWarnings:  make([]string, 0),
		RecommendedAction: "proceed",
	}

	// Basic input validation
	if !v.validateBasicInput(value, result) {
		return v.cacheAndReturn(key, result)
	}

	// Check for suspicious patterns
	if v.containsSuspiciousPatterns(value) {
		result.SecurityWarnings = append(result.SecurityWarnings,
			"Input contains potentially dangerous characters or patterns")
	}

	// Sanitize if enabled
	if v.config.SanitizeInputs {
		result.SanitizedValue = v.sanitizeString(value)
	}

	return v.cacheAndReturn(key, result)
}

// ValidateArgs checks the command line as a whole against the size limits:
// each argument at most MaxArgLength bytes, all of them together at most
// MaxTotalArgsSize bytes. It returns a validation error naming the first
// limit exceeded, nil otherwise.
//
// Orpheus does not call it: an application that wants the limits calls it
// before app.Run, typically with os.Args[1:].
func (v *InputValidator) ValidateArgs(args []string) error {
	total := 0
	for i, arg := range args {
		if len(arg) > v.config.MaxArgLength {
			return ValidationError("args", fmt.Sprintf(
				"argument %d is %d bytes, more than the %d allowed", i+1, len(arg), v.config.MaxArgLength))
		}
		total += len(arg)
		if total > v.config.MaxTotalArgsSize {
			return ValidationError("args", fmt.Sprintf(
				"arguments exceed the %d bytes allowed", v.config.MaxTotalArgsSize))
		}
	}
	return nil
}

// ValidateEnvironmentValue validates an environment variable value
//
// Used to validate environment variables that are used as defaults for CLI flags
// or that influence application behavior. Helps prevent environment poisoning attacks.
//
// Performance (BenchmarkValidateEnvironmentValue): ~160ns and 1 allocation.
func (v *InputValidator) ValidateEnvironmentValue(envName, value string) *ValidatedInput {
	result := &ValidatedInput{
		OriginalValue:     value,
		SanitizedValue:    value,
		IsValid:           true,
		ValidationErrors:  make([]string, 0),
		SecurityWarnings:  make([]string, 0),
		RecommendedAction: "proceed",
	}

	// Check if this is a trusted environment variable
	trusted := v.isEnvironmentTrusted(envName)
	if !trusted {
		result.SecurityWarnings = append(result.SecurityWarnings,
			"Environment variable from untrusted source")
	}

	// Validate the value itself
	if !v.validateBasicInput(value, result) {
		return result
	}

	// Apply stricter validation for untrusted env vars
	if !trusted && v.containsSuspiciousPatterns(value) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors,
			"Untrusted environment variable contains suspicious patterns")
		result.RecommendedAction = "reject_untrusted_env"
	}

	return result
}

// ValidateFileOperation validates that a file operation can be performed safely
//
// This method should be called before performing file operations like read, write, or execute
// to ensure the operation is safe and the user has appropriate permissions.
//
// Performance (BenchmarkValidateFileOperation): ~14us and 25 allocations
// uncached, file system access included. It calls ValidatePathFlag first, so
// with caching on -- the default -- a repeated path costs what that cache
// lookup costs.
func (v *InputValidator) ValidateFileOperation(path, operation string) *ValidatedInput {
	// First validate the path
	pathResult := v.ValidatePathFlag("file-operation", path)
	if !pathResult.IsValid {
		return pathResult
	}

	// Check specific operation permissions
	if pathResult.FilePermissions == nil {
		// Try to get file permissions if not already available
		perm, err := AnalyzeFilePermissions(pathResult.SanitizedValue, v.config.SecurityConfig)
		if err != nil {
			pathResult.IsValid = false
			pathResult.ValidationErrors = append(pathResult.ValidationErrors, err.Error())
			return pathResult
		}
		pathResult.FilePermissions = perm
	}

	// Validate specific operation
	switch strings.ToLower(operation) {
	case "read":
		if !pathResult.FilePermissions.IsReadable {
			pathResult.IsValid = false
			pathResult.ValidationErrors = append(pathResult.ValidationErrors, "File is not readable")
			pathResult.RecommendedAction = "check_permissions"
		}
	case "write":
		if pathResult.FilePermissions.IsReadOnly {
			pathResult.IsValid = false
			pathResult.ValidationErrors = append(pathResult.ValidationErrors, "File is read-only")
			pathResult.RecommendedAction = "use_readonly_mode"
		}
		if !pathResult.FilePermissions.IsWritable {
			pathResult.IsValid = false
			pathResult.ValidationErrors = append(pathResult.ValidationErrors, "File is not writable")
			pathResult.RecommendedAction = "check_permissions"
		}
	case "execute":
		if !pathResult.FilePermissions.IsExecutable {
			pathResult.IsValid = false
			pathResult.ValidationErrors = append(pathResult.ValidationErrors, "File is not executable")
			pathResult.RecommendedAction = "check_permissions"
		}

		// Warn about security risks with writable executables
		if pathResult.FilePermissions.IsWritable && pathResult.FilePermissions.IsExecutable {
			pathResult.SecurityWarnings = append(pathResult.SecurityWarnings,
				"Writable executable file poses security risk")
		}
	}

	return pathResult
}

// =============================================================================
// INTERNAL VALIDATION METHODS
// =============================================================================

// shellPatterns are rejected in every kind of input: command substitution,
// chaining and pipes, which run a command if the value ever reaches a shell.
// WHY nothing more: SQL keywords, comment markers ("--", "/*") and a lone
// "&" are ordinary in file names and free text ("insert the delete key --
// R&D") and rejected legitimate input. Keeping values out of SQL, HTML and
// shells is the job of parameterized queries and escaping, where the value
// is used.
var shellPatterns = [...]string{"$(", "`", ";", "|", "&&"}

// validateBasicInput performs basic input validation common to all input
// types: length, shell patterns, control characters.
func (v *InputValidator) validateBasicInput(value string, result *ValidatedInput) bool {
	// Check length limits
	if len(value) > v.config.MaxArgLength {
		return rejectInput(result, "Input exceeds maximum length", "reject_oversized_input")
	}

	if pattern, found := findShellPattern(value); found {
		return rejectInput(result, "Input contains dangerous pattern: "+pattern, "reject_dangerous_input")
	}

	// Check for null bytes and other control characters
	for _, r := range value {
		if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, LF, CR
			return rejectInput(result, "Input contains illegal control characters", "reject_control_chars")
		}
	}

	return true
}

// findShellPattern returns the first shell pattern contained in value.
func findShellPattern(value string) (string, bool) {
	for _, p := range shellPatterns {
		if strings.Contains(value, p) {
			return p, true
		}
	}
	return "", false
}

// rejectInput marks result invalid with one error and action; it returns
// false so callers can return it directly.
func rejectInput(result *ValidatedInput, msg, action string) bool {
	result.IsValid = false
	result.ValidationErrors = append(result.ValidationErrors, msg)
	result.RecommendedAction = action
	return false
}

// analyzeFilePermissions adds file permission analysis to validation result
func (v *InputValidator) analyzeFilePermissions(result *ValidatedInput) {
	// Only analyze if file exists
	info, err := os.Stat(result.SanitizedValue)
	if err != nil {
		return
	}

	perm, err := v.filePermissions(result.SanitizedValue, info)
	if err != nil {
		result.SecurityWarnings = append(result.SecurityWarnings,
			"Could not analyze file permissions: "+err.Error())
		return
	}
	result.FilePermissions = perm

	// Add security warnings based on permission analysis
	switch perm.SecurityRisk {
	case "critical", "high":
		result.SecurityWarnings = append(result.SecurityWarnings,
			"File has high security risk: "+perm.SecurityRisk)
	case "medium":
		result.SecurityWarnings = append(result.SecurityWarnings,
			"File has medium security risk")
	}
}

// filePermissions analyzes a path that exists.
// WHY: when ValidatePathArgs is on, the path has just passed
// ValidateSecurePath with this same SecurityConfig; AnalyzeFilePermissions
// would run it again, symlink resolution included, for the same answer.
func (v *InputValidator) filePermissions(path string, info os.FileInfo) (*FilePermission, error) {
	if v.config.ValidatePathArgs {
		return analyzeFileInfo(path, info), nil
	}
	return AnalyzeFilePermissions(path, v.config.SecurityConfig)
}

// suspiciousChars are shell metacharacters that are not rejected outright:
// they earn a warning on string flags and reject untrusted environment
// values. "&" is not one: alone it is ordinary text, and "&&" is rejected.
const suspiciousChars = ";|`$(){}[]<>"

// containsSuspiciousPatterns checks for patterns that might indicate
// injection attempts: shell metacharacters, "../" or "..\\", and
// percent-escapes.
// WHY not a regexp: a byte scan answers the same question several times
// faster and without allocating.
func (v *InputValidator) containsSuspiciousPatterns(value string) bool {
	return strings.ContainsAny(value, suspiciousChars) ||
		strings.Contains(value, "../") || strings.Contains(value, `..\`) ||
		containsPercentEscape(value)
}

// containsPercentEscape reports a "%" followed by two hex digits.
func containsPercentEscape(value string) bool {
	for i := strings.IndexByte(value, '%'); i >= 0 && i+2 < len(value); {
		if _, ok := unhexPair(value[i+1], value[i+2]); ok {
			return true
		}
		next := strings.IndexByte(value[i+1:], '%')
		if next < 0 {
			return false
		}
		i += next + 1
	}
	return false
}

// sanitizeString removes or escapes potentially dangerous characters
func (v *InputValidator) sanitizeString(value string) string {
	// Remove null bytes
	value = strings.ReplaceAll(value, "\x00", "")

	// Remove other problematic control characters
	var sanitized strings.Builder
	sanitized.Grow(len(value))

	for _, r := range value {
		// Keep printable characters and common whitespace
		if r >= 32 || r == 9 || r == 10 || r == 13 {
			sanitized.WriteRune(r)
		}
	}

	return sanitized.String()
}

// isEnvironmentTrusted reports whether envName is trusted: equal to an
// entry of TrustedEnvPrefixes, or starting with an entry that ends in "_".
// WHY: a bare prefix match let "USER" trust USER_EVIL and USERNAME, and
// "PATH" trust PATHEXT. An entry is a variable name unless it is written as
// a namespace ("MYAPP_").
func (v *InputValidator) isEnvironmentTrusted(envName string) bool {
	for _, entry := range v.config.TrustedEnvPrefixes {
		if envName == entry || (strings.HasSuffix(entry, "_") && strings.HasPrefix(envName, entry)) {
			return true
		}
	}
	return false
}

// Cache key kinds: one per validation method.
// WHY: ValidatePathFlag and ValidateStringFlag shared "flag:value" keys, so a
// value validated as a string came back from the cache as a valid path,
// with no path checks run.
const (
	cacheKindPath   = "p\x00"
	cacheKindString = "s\x00"
)

// cacheKey builds the cache key for one validation. NUL separates the parts
// because a flag name may contain ':' and a NUL in value is always rejected.
func cacheKey(kind, flagName, value string) string {
	return kind + flagName + "\x00" + value
}

// lookupCache returns the cache key and the cached result, if any.
// WHY: with caching off the key is never built; concatenating it costs an
// allocation the size of flag name plus value on every call.
func (v *InputValidator) lookupCache(kind, flagName, value string) (string, *ValidatedInput) {
	if !v.config.EnableCaching {
		return "", nil
	}
	key := cacheKey(kind, flagName, value)
	return key, v.getCachedResult(key)
}

// getCachedResult retrieves a cached validation result
func (v *InputValidator) getCachedResult(key string) *ValidatedInput {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()

	if result, exists := v.cache[key]; exists {
		// Return a copy to prevent modification of cached result
		return v.copyValidatedInput(result)
	}

	return nil
}

// cacheAndReturn caches the result and returns it
func (v *InputValidator) cacheAndReturn(key string, result *ValidatedInput) *ValidatedInput {
	if v.config.EnableCaching {
		v.cacheMu.Lock()
		defer v.cacheMu.Unlock()

		// Implement simple LRU by clearing cache when it gets too large
		if len(v.cache) >= v.config.CacheSize {
			v.cache = make(map[string]*ValidatedInput)
		}

		v.cache[key] = v.copyValidatedInput(result)
	}

	return result
}

// copyValidatedInput creates a deep copy of ValidatedInput
func (v *InputValidator) copyValidatedInput(original *ValidatedInput) *ValidatedInput {
	result := &ValidatedInput{
		OriginalValue:     original.OriginalValue,
		SanitizedValue:    original.SanitizedValue,
		IsValid:           original.IsValid,
		RecommendedAction: original.RecommendedAction,
		FilePermissions:   original.FilePermissions, // Shallow copy is OK for read-only data
	}

	// Deep copy slices
	result.ValidationErrors = make([]string, len(original.ValidationErrors))
	result.SecurityWarnings = make([]string, len(original.SecurityWarnings))
	copy(result.ValidationErrors, original.ValidationErrors)
	copy(result.SecurityWarnings, original.SecurityWarnings)

	return result
}

// ClearCache clears the validation cache
func (v *InputValidator) ClearCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cache = make(map[string]*ValidatedInput)
}

// GetCacheStats returns cache statistics for monitoring
func (v *InputValidator) GetCacheStats() map[string]int {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()

	return map[string]int{
		"cache_size":     len(v.cache),
		"cache_capacity": v.config.CacheSize,
	}
}

// =============================================================================
// CONVENIENCE METHODS FOR COMMON CLI PATTERNS
// =============================================================================

// ValidateFileFlag validates a file flag (common pattern: --file, --config, --input)
func (v *InputValidator) ValidateFileFlag(value string) *ValidatedInput {
	result := v.ValidatePathFlag("file", value)

	// Additional checks for files specifically
	if result.IsValid && result.FilePermissions == nil {
		if _, err := os.Stat(result.SanitizedValue); err != nil {
			result.ValidationErrors = append(result.ValidationErrors, "File does not exist")
			result.RecommendedAction = "check_file_exists"
			result.IsValid = false
		}
	}

	return result
}

// ValidateDirectoryFlag validates a directory flag (common pattern: --dir, --output-dir)
func (v *InputValidator) ValidateDirectoryFlag(value string) *ValidatedInput {
	result := v.ValidatePathFlag("directory", value)

	// Additional checks for directories specifically
	if result.IsValid {
		if info, err := os.Stat(result.SanitizedValue); err == nil {
			if !info.IsDir() {
				result.ValidationErrors = append(result.ValidationErrors, "Path is not a directory")
				result.RecommendedAction = "use_directory_path"
				result.IsValid = false
			}
		} else {
			result.SecurityWarnings = append(result.SecurityWarnings, "Directory does not exist")
		}
	}

	return result
}

// ValidateOutputFlag validates an output file flag with write permission check
func (v *InputValidator) ValidateOutputFlag(value string) *ValidatedInput {
	result := v.ValidatePathFlag("output", value)

	if result.IsValid {
		// Check if we can write to this location
		dir := filepath.Dir(result.SanitizedValue)
		if info, err := os.Stat(dir); err != nil {
			result.ValidationErrors = append(result.ValidationErrors, "Output directory does not exist")
			result.RecommendedAction = "create_output_directory"
			result.IsValid = false
		} else if !info.IsDir() {
			result.ValidationErrors = append(result.ValidationErrors, "Output parent is not a directory")
			result.RecommendedAction = "check_output_path"
			result.IsValid = false
		}

		// If file exists, check if it's writable
		if result.IsValid {
			if writeResult := v.ValidateFileOperation(result.SanitizedValue, "write"); !writeResult.IsValid {
				// File exists but not writable
				result.ValidationErrors = append(result.ValidationErrors, writeResult.ValidationErrors...)
				result.SecurityWarnings = append(result.SecurityWarnings, writeResult.SecurityWarnings...)
				result.RecommendedAction = writeResult.RecommendedAction
				result.IsValid = false
			}
		}
	}

	return result
}

// String returns a human-readable representation of validation result
func (result *ValidatedInput) String() string {
	var parts []string

	parts = append(parts, "Original: "+result.OriginalValue)
	if result.SanitizedValue != result.OriginalValue {
		parts = append(parts, "Sanitized: "+result.SanitizedValue)
	}

	if result.IsValid {
		parts = append(parts, "Status: VALID")
	} else {
		parts = append(parts, "Status: INVALID")
	}

	if len(result.ValidationErrors) > 0 {
		parts = append(parts, "Errors: "+strings.Join(result.ValidationErrors, ", "))
	}

	if len(result.SecurityWarnings) > 0 {
		parts = append(parts, "Warnings: "+strings.Join(result.SecurityWarnings, ", "))
	}

	if result.RecommendedAction != "proceed" {
		parts = append(parts, "Action: "+result.RecommendedAction)
	}

	return strings.Join(parts, "\n")
}

// IsSecure returns true if the input has no security issues
func (result *ValidatedInput) IsSecure() bool {
	return result.IsValid && len(result.SecurityWarnings) == 0
}

// ValidateInput performs general input validation for security testing.
//
// This method provides a generic validation interface for CLI input security testing,
// checking for common attack patterns and malicious input. It returns an error
// if the input fails validation, making it suitable for security test assertions.
//
// Parameters:
//   - input: The input string to validate
//
// Returns:
//   - error: nil if input is valid, error describing the security issue if invalid
func (v *InputValidator) ValidateInput(input string) error {
	// Use ValidateStringFlag for generic input validation
	result := v.ValidateStringFlag("generic", input)

	if !result.IsValid {
		// Create error from validation result
		var errorMessages []string

		// Add validation errors
		errorMessages = append(errorMessages, result.ValidationErrors...)

		// Add security warnings as errors for security testing
		if len(result.SecurityWarnings) > 0 {
			errorMessages = append(errorMessages, result.SecurityWarnings...)
		}

		if len(errorMessages) > 0 {
			return ValidationError("input",
				"Input validation failed: "+strings.Join(errorMessages, "; "))
		}

		return ValidationError("input", "Input validation failed")
	}

	// Additional check for security warnings even if technically "valid"
	if len(result.SecurityWarnings) > 0 {
		return ValidationError("input",
			"Security concerns detected: "+strings.Join(result.SecurityWarnings, "; "))
	}

	return nil
}
