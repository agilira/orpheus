// Package security provides the input and path validation of Orpheus:
// InputValidator for flags, arguments and environment values,
// ValidateSecurePath and the file permission checks. Its controls are
// specifically designed for CLI applications with focus on:
// - Path validation and traversal prevention
// - File permissions analysis and enforcement
// - Input sanitization and validation
// - Environment variable security
//
// SECURITY PHILOSOPHY:
// - Defense in depth with multiple validation layers
// - Zero-trust approach to all external input
// - Performance-conscious security (no impact on CLI responsiveness)
// - Clear, actionable error messages for developers
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode"

	"github.com/agilira/orpheus/errs"
)

// SecurityConfig holds security-related configuration options
type Config struct {
	// EnablePathValidation enables comprehensive path security validation
	EnablePathValidation bool
	// MaxPathLength limits the maximum allowed path length (default: 4096)
	MaxPathLength int
	// MaxPathDepth limits directory traversal depth (default: 50)
	MaxPathDepth int
	// AllowedPaths restricts operations to these directories and what lies
	// under them, matched on whole path segments; "*" matches one segment.
	// Relative entries are resolved against the working directory at
	// validation time, as relative paths are.
	AllowedPaths []string
	// DeniedPaths explicitly denies these directories and what lies under
	// them; matching and relative entries work as for AllowedPaths.
	DeniedPaths []string
	// EnableFilePermissionCheck enables file permission analysis
	EnableFilePermissionCheck bool
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultConfig() Config {
	return Config{
		EnablePathValidation:      true,
		MaxPathLength:             4096,
		MaxPathDepth:              50,
		AllowedPaths:              []string{},
		DeniedPaths:               getDefaultDeniedPaths(),
		EnableFilePermissionCheck: true,
	}
}

// FilePermission represents detailed file permission information
type FilePermission struct {
	Path           string      // Full path to the file
	Mode           os.FileMode // Raw file mode
	IsReadable     bool        // Current user can read
	IsWritable     bool        // Current user can write
	IsExecutable   bool        // Current user can execute
	IsReadOnly     bool        // File is marked read-only
	IsDirectory    bool        // Path is a directory
	Size           int64       // File size in bytes
	ModTime        int64       // Last modification time (Unix timestamp)
	Owner          string      // File owner (best effort, platform dependent)
	Group          string      // File group (best effort, platform dependent)
	SecurityRisk   string      // Security risk assessment
	Recommendation string      // Security recommendation
}

// PathSecurityResult represents the result of path security validation
type PathSecurityResult struct {
	Path           string   // Original path being validated
	IsValid        bool     // Whether the path passes security validation
	NormalizedPath string   // Cleaned and normalized version of the path
	Errors         []string // List of security violations found
	Warnings       []string // List of security warnings
	Risk           string   // Risk level: "low", "medium", "high", "critical"
}

// =============================================================================
// PATH VALIDATION AND SECURITY
// =============================================================================

// ValidateSecurePath performs comprehensive path security validation
// similar to Argus's security model but adapted for CLI applications.
//
// SECURITY LAYERS:
// 1. Empty/null path rejection
// 2. Path length and complexity limits
// 3. Directory traversal pattern detection, percent-encoded forms included
// 4. System file/directory protection
// 5. Control character filtering
// 6. Windows device name and alternate data stream protection (Windows only)
// 7. Allowed/denied lists, matched on whole path segments ("*" allowed),
// with relative paths and entries made absolute from the working directory
// 8. The same system and list checks on the symlink-resolved path
// 9. Case-insensitive security validation
//
// Performance (BenchmarkValidateSecurePath, AMD Ryzen 5 7520U): ~350ns and 2
// allocations for a path that is rejected; a path that passes also pays one
// lstat per component, and a relative one a getcwd. Rejection is the cheaper case, not the more expensive one: a
// violation returns at the layer that found it, while a valid path is carried
// through every remaining check.
func ValidateSecurePath(path string, config Config) *PathSecurityResult {
	result := &PathSecurityResult{
		Path:           path,
		IsValid:        true,
		NormalizedPath: path,
		Errors:         make([]string, 0),
		Warnings:       make([]string, 0),
		Risk:           "low",
	}

	// Layer 1: Empty path validation
	if path == "" {
		result.addError("empty path not allowed")
		return result
	}

	// Layer 2: Path length validation
	if len(path) > config.MaxPathLength {
		result.addError(fmt.Sprintf("path too long (max %d characters)", config.MaxPathLength))
		return result
	}

	// Layer 3: Clean and normalize path for further validation
	cleanPath := filepath.Clean(path)
	result.NormalizedPath = cleanPath

	// Layers 4-8 read the path as spelled; the first violation ends validation.
	for _, layer := range spellingLayers {
		if result.fail(layer(path, cleanPath, config)) {
			return result
		}
	}

	// Layers 9-11 judge where the path is, so they need it absolute.
	loc, err := locatePath(cleanPath, config)
	if err != nil {
		result.fail("cannot resolve relative path: "+err.Error(), "high")
		return result
	}
	for _, layer := range locationLayers {
		if result.fail(layer(loc, config)) {
			return result
		}
	}

	return result
}

// fail records msg (when not empty) and risk (when not empty); it reports
// whether validation must stop.
func (result *PathSecurityResult) fail(msg, risk string) bool {
	if msg == "" {
		return false
	}
	result.addError(msg)
	if risk != "" {
		result.Risk = risk
	}
	return true
}

// pathLocation is a cleaned path in absolute form, and the working
// directory used to make it so ("" when none was needed).
type pathLocation struct {
	typed string // the cleaned path as given, relative or not
	abs   string
	lower string // abs in lowerSlash form, computed once for every check
	cwd   string
}

// locatePath makes clean absolute, reading the working directory at most
// once, and only when the path or an allowed/denied entry is relative.
// WHY: "passwd" typed in /etc, or "etc/hostname" typed in /, matched none of
// the absolute system, allowed or denied entries and was accepted.
func locatePath(clean string, config Config) (pathLocation, error) {
	if isAbsolutePath(clean) && !hasRelativeEntry(config) {
		return pathLocation{typed: clean, abs: clean, lower: lowerSlash(clean)}, nil
	}
	cwd, err := workingDir()
	if err != nil {
		return pathLocation{}, err
	}
	abs := absoluteFrom(clean, cwd)
	return pathLocation{typed: clean, abs: abs, lower: lowerSlash(abs), cwd: cwd}, nil
}

// isAbsolutePath is filepath.IsAbs that also counts a Windows drive path
// ("C:\\x", "C:/x") as absolute on every OS.
// WHY: the default denied list carries Windows entries everywhere; joined to
// a Unix working directory they would mean nothing, and a typed C:\\Windows
// path is compared as spelled, as before.
func isAbsolutePath(p string) bool {
	if filepath.IsAbs(p) {
		return true
	}
	return len(p) >= 3 && p[1] == ':' && isASCIILetter(p[0]) && (p[2] == '\\' || p[2] == '/')
}

func isASCIILetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func hasRelativeEntry(config Config) bool {
	for _, lists := range [...][]string{config.AllowedPaths, config.DeniedPaths} {
		for _, entry := range lists {
			if !isAbsolutePath(entry) {
				return true
			}
		}
	}
	return false
}

// absoluteFrom resolves p against cwd unless it is already absolute.
func absoluteFrom(p, cwd string) string {
	if isAbsolutePath(p) {
		return p
	}
	if runtime.GOOS == "windows" {
		// WHY: a rooted path without a drive ("\\x") or a drive-relative one
		// ("C:x") does not resolve by joining; Abs asks the system.
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	return filepath.Join(cwd, p)
}

// pathLayer checks one property of a path. raw is the path as given, clean
// its filepath.Clean form. It returns an error message ("" = passed) and the
// risk level to record ("" = leave it unchanged).
// WHY config by value: a pointer through this indirect call escapes, and
// that cost one heap allocation per ValidateSecurePath.
type pathLayer func(raw, clean string, config Config) (msg, risk string)

// locationLayer checks where an absolute path points.
type locationLayer func(loc pathLocation, config Config) (msg, risk string)

// spellingLayers and locationLayers list the checks of ValidateSecurePath
// in the order they run.
// WHY tables: one loop each keeps ValidateSecurePath simple however many
// layers there are, and the order stays visible in one place. The spelling
// layers run first, so a rejected path never pays for the working directory.
var spellingLayers = [...]pathLayer{
	checkTraversal,    // Layer 4: traversal, plain or percent-encoded
	checkDepth,        // Layer 5: path depth
	checkControlChars, // Layer 6: control characters
	checkDeviceName,   // Layer 7: Windows device names
	checkDataStream,   // Layer 8: Windows alternate data streams
}

var locationLayers = [...]locationLayer{
	checkSystemPath,    // Layer 9: system directories
	checkAllowedDenied, // Layer 10: allowed/denied lists, and where links lead
}

func checkTraversal(_, clean string, _ Config) (string, string) {
	if containsTraversalPattern(clean) || containsEncodedTraversal(clean) {
		return "dangerous traversal pattern detected", "critical"
	}
	return "", ""
}

func checkDepth(_, clean string, config Config) (string, string) {
	if strings.Count(clean, string(os.PathSeparator)) > config.MaxPathDepth {
		return fmt.Sprintf("path too complex (max depth %d)", config.MaxPathDepth), ""
	}
	return "", ""
}

func checkControlChars(raw, _ string, _ Config) (string, string) {
	if containsControlCharacters(raw) {
		return "control characters in path not allowed", "high"
	}
	return "", ""
}

func checkDeviceName(raw, _ string, _ Config) (string, string) {
	if isWindowsDeviceName(filepath.Base(raw)) {
		return "windows device names not allowed", "high"
	}
	return "", ""
}

func checkSystemPath(loc pathLocation, _ Config) (string, string) {
	if isSystemPathLower(loc.lower) {
		return "system file/directory access not allowed", "critical"
	}
	return "", ""
}

// checkDataStream rejects NTFS alternate data streams.
// WHY Windows only: elsewhere ':' is an ordinary filename character
// (timestamps, host:port), and rejecting it refused legitimate files.
func checkDataStream(raw, _ string, _ Config) (string, string) {
	if runtime.GOOS == "windows" && containsAlternateDataStream(raw) {
		return "alternate data streams not allowed", "high"
	}
	return "", ""
}

// checkAllowedDenied judges the path both as spelled and in canonical form
// (links resolved, entries too).
// WHY both: the spelling misses links in the path (./cfg -> /etc) and
// entries reached through an alias (/tmp/alias, /etc on macOS being
// /private/etc); the canonical form alone would miss entries that do not
// exist yet. A path through a link must be allowed in canonical form: its
// spelling says nothing about where it leads.
func checkAllowedDenied(loc pathLocation, config Config) (string, string) {
	canon := canonicalize(loc)
	if canon.viaLink && isSystemPathLower(canon.lower) {
		return linkForbidden()
	}
	if !isPathAllowed(loc, canon, config) {
		return forbidden(canon, "path not in allowed list")
	}
	if isPathDenied(loc, canon, config) {
		return forbidden(canon, "path explicitly denied")
	}
	return "", ""
}

func linkForbidden() (string, string) {
	return "symlink resolves to a forbidden path", "critical"
}

// forbidden names the spelling's fault, or the link's when one is involved.
func forbidden(canon canonicalPath, msg string) (string, string) {
	if canon.viaLink {
		return linkForbidden()
	}
	return msg, "high"
}

// canonicalPath is a path with its symlinks resolved.
type canonicalPath struct {
	lower   string // lowerSlash form
	viaLink bool   // the path went through at least one link
}

// canonicalize resolves loc's links, only when it has one.
// A link loop cannot be resolved; the path is then judged as spelled, and
// opening it fails anyway.
func canonicalize(loc pathLocation) canonicalPath {
	if !containsSymlink(symlinkWalkPath(loc)) {
		return canonicalPath{lower: loc.lower}
	}
	resolved, _, ok := resolvePhysical(loc.abs)
	if !ok || resolved == loc.abs {
		return canonicalPath{lower: loc.lower}
	}
	return canonicalPath{lower: lowerSlash(resolved), viaLink: true}
}

// symlinkWalkPath is the part of loc that may hold a link.
// WHY: where workingDir is the kernel's physical directory (Unix), only the
// typed part of a relative path can contain one, and walking the working
// directory too would cost one lstat per component for nothing.
func symlinkWalkPath(loc pathLocation) string {
	if physicalWorkingDir {
		return loc.typed
	}
	return loc.abs
}

// containsSymlink reports whether any existing component of p is a symlink.
// WHY: one Lstat per component and no allocation; the full resolution runs
// only when a link is found. The walk stops at the first component that
// does not exist, since nothing below it can be a link.
func containsSymlink(p string) bool {
	start := len(filepath.VolumeName(p)) + 1
	for i := start; i <= len(p); i++ {
		if i < len(p) && !os.IsPathSeparator(p[i]) {
			continue
		}
		info, err := os.Lstat(p[:i])
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

// maxLinkHops bounds the links followed by resolvePhysical, as the kernel
// bounds symlink loops (ELOOP).
const maxLinkHops = 40

// resolvePhysical resolves the absolute path p one component at a time, as
// the kernel does: a link is replaced by its target, ".." goes to the parent
// of the directory actually reached, and from the first missing component
// on the rest is appended as is. complete is false when something is
// missing; ok is false on a link loop.
// WHY not EvalSymlinks: it fails on a dangling link or a missing tail, and
// the lexical fallbacks that filled the gap collapsed ".." against the
// link's own directory: allowed/jump/../x is not allowed/x when jump is a
// link. Writing through a dangling link creates its target, so the target
// is what must be judged.
func resolvePhysical(p string) (resolved string, complete, ok bool) {
	vol := filepath.VolumeName(p)
	resolved, rest := vol+string(filepath.Separator), p[len(vol):]
	complete = true
	for hops := 0; rest != ""; {
		var comp string
		comp, rest = cutComponent(rest)
		next, target, isLink := stepPhysical(resolved, comp, complete)
		switch {
		case isLink:
			if hops++; hops > maxLinkHops {
				return p, false, false
			}
			resolved, rest = linkRestart(resolved, target, rest)
		case next == "":
			complete = false
			resolved = filepath.Join(resolved, comp)
		default:
			resolved = next
		}
	}
	return resolved, complete, true
}

// stepPhysical moves from the physical directory dir by one component.
// It returns the next physical path, or the link target to follow, or ""
// when comp does not exist (or anything before it did not).
func stepPhysical(dir, comp string, exists bool) (next, target string, isLink bool) {
	switch {
	case comp == "" || comp == ".":
		return dir, "", false
	case comp == "..":
		return filepath.Dir(dir), "", false
	case !exists:
		return "", "", false
	}
	next = filepath.Join(dir, comp)
	info, err := os.Lstat(next)
	if err != nil {
		return "", "", false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return next, "", false
	}
	target, err = os.Readlink(next)
	if err != nil {
		return "", "", false
	}
	return "", target, true
}

// linkRestart continues resolution with a link's target in front of the
// remaining components: from the root for an absolute target, from the
// link's directory otherwise.
func linkRestart(dir, target, rest string) (string, string) {
	if rest != "" {
		target += string(filepath.Separator) + rest
	}
	if filepath.IsAbs(target) {
		vol := filepath.VolumeName(target)
		return vol + string(filepath.Separator), target[len(vol):]
	}
	if len(target) > 0 && os.IsPathSeparator(target[0]) {
		return filepath.VolumeName(dir) + string(filepath.Separator), target[1:]
	}
	return dir, target
}

// cutComponent splits off the first path component.
func cutComponent(p string) (comp, rest string) {
	for i := 0; i < len(p); i++ {
		if os.IsPathSeparator(p[i]) {
			return p[:i], p[i+1:]
		}
	}
	return p, ""
}

// canonicalEntries memoizes the canonical form of allowed/denied entries
// that fully exist, keyed by their absolute spelling.
// WHY: every accepted path is compared with every entry in canonical form,
// and resolving fifteen default entries each time would cost dozens of
// system calls. Only complete resolutions are kept, so an entry that does
// not exist yet is resolved again until it does. Bounded: cleared when full.
var canonicalEntries = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

const maxCanonicalEntries = 1024

// canonicalEntry returns entry, made absolute against cwd, in canonical
// form. For a glob entry only the part before the first glob segment is
// resolved. Windows drive paths on other systems are returned as spelled.
func canonicalEntry(entry, cwd string) string {
	abs := absoluteFrom(entry, cwd)
	if !filepath.IsAbs(abs) {
		return abs
	}

	canonicalEntries.RLock()
	canon, found := canonicalEntries.m[abs]
	canonicalEntries.RUnlock()
	if found {
		return canon
	}

	static, glob := splitGlob(abs)
	canon, complete, ok := resolvePhysical(static)
	if !ok {
		return abs
	}
	canon = filepath.Join(canon, glob)
	if complete {
		rememberCanonical(abs, canon)
	}
	return canon
}

func rememberCanonical(abs, canon string) {
	canonicalEntries.Lock()
	defer canonicalEntries.Unlock()
	if len(canonicalEntries.m) >= maxCanonicalEntries {
		canonicalEntries.m = make(map[string]string)
	}
	canonicalEntries.m[abs] = canon
}

// splitGlob splits a cleaned path before its first segment holding a glob
// character: "/home/*/.ssh" gives "/home" and "*/.ssh".
func splitGlob(p string) (static, glob string) {
	i := strings.IndexAny(p, "*?[")
	if i < 0 {
		return p, ""
	}
	cut := strings.LastIndexAny(p[:i], `/\`)
	if cut <= len(filepath.VolumeName(p)) {
		return p[:cut+1], p[cut+1:]
	}
	return p[:cut], p[cut+1:]
}

// =============================================================================
// FILE PERMISSION ANALYSIS
// =============================================================================

// AnalyzeFilePermissions provides comprehensive file permission analysis
// for security assessment and CLI operation planning.
//
// SECURITY BENEFITS:
// - Detects read-only files before write operations
// - Identifies potential privilege escalation vectors
// - Provides security risk assessment for file operations
// - Enables proactive permission management
//
// Performance (BenchmarkAnalyzeFilePermissions): ~13us and 22 allocations,
// file system access (symlink walk, stat, access(2)) included.
func AnalyzeFilePermissions(path string, config Config) (*FilePermission, error) {
	// First validate the path if security is enabled
	if config.EnablePathValidation {
		pathResult := ValidateSecurePath(path, config)
		if !pathResult.IsValid {
			return nil, errs.ValidationError("file-permission",
				fmt.Sprintf("insecure path: %s", strings.Join(pathResult.Errors, ", ")))
		}
		path = pathResult.NormalizedPath
	}

	// WHY Stat, not Lstat: the path has been validated through its symlinks,
	// so describe the file that will actually be opened; a link's own mode
	// is always 0777 and made every symlink look world-writable.
	info, err := os.Stat(path)
	if err != nil {
		return nil, errs.ExecutionError("file-permission",
			fmt.Sprintf("cannot access file: %v", err))
	}

	return analyzeFileInfo(path, info), nil
}

// analyzeFileInfo builds the permission report for an already validated path.
func analyzeFileInfo(path string, info os.FileInfo) *FilePermission {
	perm := &FilePermission{
		Path:        path,
		Mode:        info.Mode(),
		IsDirectory: info.IsDir(),
		Size:        info.Size(),
		ModTime:     info.ModTime().Unix(),
	}

	// Analyze permissions based on file mode
	perm.analyzeBasicPermissions()

	// Perform platform-specific analysis
	perm.analyzePlatformSpecific()

	// Assess security risks
	perm.assessSecurityRisk()

	// Generate security recommendations
	perm.generateRecommendations()

	return perm
}

// CheckFileReadable efficiently checks if a file is readable by current user
func CheckFileReadable(path string, config Config) (bool, error) {
	perm, err := AnalyzeFilePermissions(path, config)
	if err != nil {
		return false, err
	}
	return perm.IsReadable, nil
}

// CheckFileWritable efficiently checks if a file is writable by current user
func CheckFileWritable(path string, config Config) (bool, error) {
	perm, err := AnalyzeFilePermissions(path, config)
	if err != nil {
		return false, err
	}
	return perm.IsWritable, nil
}

// CheckFileExecutable efficiently checks if a file is executable by current user
func CheckFileExecutable(path string, config Config) (bool, error) {
	perm, err := AnalyzeFilePermissions(path, config)
	if err != nil {
		return false, err
	}
	return perm.IsExecutable, nil
}

// =============================================================================
// INTERNAL SECURITY VALIDATION FUNCTIONS
// =============================================================================

// containsTraversalPattern detects directory traversal patterns (case-insensitive)
func containsTraversalPattern(path string) bool {
	// Convert to lowercase for case-insensitive matching
	lowerPath := strings.ToLower(path)

	// Common traversal patterns
	// Note: We check the path AFTER filepath.Clean() which normalizes "./" to "."
	// so we don't need to explicitly check for "./" patterns
	patterns := []string{
		"..",         // Parent directory marker
		"../",        // Unix/Linux traversal
		"..\\",       // Windows traversal
		"/..",        // Traversal with leading slash
		"\\..",       // Windows traversal with leading backslash
		"%2e%2e",     // URL-encoded .. (%2e = .)
		"%252e%252e", // Double URL-encoded ..
	}

	for _, pattern := range patterns {
		if strings.Contains(lowerPath, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// containsEncodedTraversal decodes percent-encoding, up to three layers
// deep, and looks for ".." in the result.
// WHY: the literal "%2e%2e" pattern missed mixed forms such as "%2e." and
// ".%2E", which decode to the same "..".
func containsEncodedTraversal(path string) bool {
	for range 3 {
		if strings.IndexByte(path, '%') < 0 {
			return false
		}
		path = percentDecode(path)
		if strings.Contains(path, "..") {
			return true
		}
	}
	return false
}

// percentDecode decodes every valid %XX escape and keeps anything else as is.
// WHY not url.PathUnescape: it rejects the whole string on one bad escape,
// so "%zz%2e." would hide the "..".
func percentDecode(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			if v, ok := unhexPair(s[i+1], s[i+2]); ok {
				b.WriteByte(v)
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func unhexPair(hi, lo byte) (byte, bool) {
	h, ok1 := unhex(hi)
	l, ok2 := unhex(lo)
	return h<<4 | l, ok1 && ok2
}

func unhex(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// containsControlCharacters detects control characters that could be used for injection
func containsControlCharacters(path string) bool {
	for _, r := range path {
		// Check for null bytes and other control characters
		if r < 32 || r == 127 || (r >= 128 && r <= 159) {
			return true
		}
	}
	return false
}

// isWindowsDeviceName checks for Windows reserved device names
func isWindowsDeviceName(name string) bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// Convert to uppercase for comparison
	upperName := strings.ToUpper(name)

	// Remove extension if present
	if dotIndex := strings.LastIndex(upperName, "."); dotIndex != -1 {
		upperName = upperName[:dotIndex]
	}

	deviceNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, device := range deviceNames {
		if upperName == device {
			return true
		}
	}

	return false
}

// lowerSlash is the form every path comparison uses: lower case, forward
// slashes where the OS separator is a backslash.
func lowerSlash(path string) string {
	return strings.ToLower(filepath.ToSlash(path))
}

// isSystemPathLower reports whether a path, already in lowerSlash form,
// is in a system directory.
func isSystemPathLower(lowerPath string) bool {
	// Unix/Linux system paths
	unixSystemPaths := []string{
		"/etc/", "/proc/", "/sys/", "/dev/", "/boot/", "/root/",
		"/usr/bin/", "/usr/sbin/", "/sbin/", "/bin/",
	}

	// Windows system paths
	windowsSystemPaths := []string{
		"c:/windows/", "c:/program files/", "c:/program files (x86)/",
		"c:/users/default/", "c:/programdata/",
	}

	allPaths := append(unixSystemPaths, windowsSystemPaths...)

	for _, sysPath := range allPaths {
		if strings.HasPrefix(lowerPath, sysPath) {
			return true
		}
	}

	// Check for specific sensitive files (case-insensitive)
	sensitiveFiles := []string{
		"/etc/passwd", "/etc/shadow", "/etc/hosts", "/etc/sudoers",
		"c:/windows/system32/config/sam", "c:/windows/system32/config/security",
	}

	for _, sensitive := range sensitiveFiles {
		if strings.HasSuffix(lowerPath, strings.ToLower(sensitive)) {
			return true
		}
	}

	return false
}

// containsAlternateDataStream detects Windows Alternate Data Stream notation
func containsAlternateDataStream(path string) bool {
	// Convert path to use forward slashes for consistent parsing
	normalizedPath := filepath.ToSlash(path)

	// Look for colon in filename (not drive letter)
	parts := strings.Split(normalizedPath, "/")
	if len(parts) == 0 {
		return false
	}

	// Check each part for ADS, but skip the first part if it looks like a drive
	for i, part := range parts {
		// Skip drive letters (C:, D:, etc.) in the first part
		if i == 0 && len(part) >= 2 && part[1] == ':' && unicode.IsLetter(rune(part[0])) {
			// This is a drive letter, check if there's anything after the colon
			if len(part) > 2 {
				// There's something after C:, check for additional colons
				remainingPart := part[2:]
				if strings.Contains(remainingPart, ":") {
					return true // ADS detected after drive letter
				}
			}
			continue
		}

		// For all other parts, check for any colon (indicates ADS)
		if strings.Contains(part, ":") {
			return true
		}
	}

	return false
}

// isPathAllowed checks if path is in allowed list (if configured).
// Relative entries are resolved against loc.cwd, like the path. A path that
// goes through a link is allowed only by its canonical form.
func isPathAllowed(loc pathLocation, canon canonicalPath, config Config) bool {
	if len(config.AllowedPaths) == 0 {
		return true // No restrictions if no allowed paths configured
	}

	for _, allowed := range config.AllowedPaths {
		if matchesEntry(loc, canon, allowed, !canon.viaLink) {
			return true
		}
	}

	return false
}

// isPathDenied checks if path is explicitly denied, as spelled or in
// canonical form. Relative entries are resolved against loc.cwd.
func isPathDenied(loc pathLocation, canon canonicalPath, config Config) bool {
	for _, denied := range config.DeniedPaths {
		if matchesEntry(loc, canon, denied, true) {
			return true
		}
	}

	return false
}

// matchesEntry matches entry against the canonical path and, when
// spellingCounts, against the path as spelled.
// WHY the skip: when neither the path nor the entry goes through a link the
// two comparisons are the same one, and each costs a lower-casing.
func matchesEntry(loc pathLocation, canon canonicalPath, entry string, spellingCounts bool) bool {
	spelled := absoluteFrom(entry, loc.cwd)
	if spellingCounts && matchPathEntry(loc.lower, spelled) {
		return true
	}
	canonEntry := canonicalEntry(entry, loc.cwd)
	if spellingCounts && !canon.viaLink && canonEntry == spelled {
		return false
	}
	return matchPathEntry(canon.lower, canonEntry)
}

// matchPathEntry reports whether lowerPath (lower case, forward slashes) is
// entry or lies under it. An entry with glob characters ("/home/*/.ssh") is
// matched segment by segment.
// WHY: a bare string prefix let /srv/app allow /srv/app-secrets, made /dev
// deny /development, and compared the "*" of /home/*/.ssh literally.
func matchPathEntry(lowerPath, entry string) bool {
	lowerEntry := strings.ToLower(filepath.ToSlash(entry))
	if strings.ContainsAny(lowerEntry, "*?[") {
		return matchGlobPrefix(lowerPath, strings.TrimSuffix(lowerEntry, "/"))
	}
	return hasPathPrefix(lowerPath, lowerEntry)
}

// hasPathPrefix is strings.HasPrefix that only matches on a path boundary.
// Backslash counts as a separator too, for Windows entries checked elsewhere.
func hasPathPrefix(p, prefix string) bool {
	if !strings.HasPrefix(p, prefix) {
		return false
	}
	if len(p) == len(prefix) || prefix == "" {
		return true
	}
	return isSeparatorByte(prefix[len(prefix)-1]) || isSeparatorByte(p[len(prefix)])
}

func isSeparatorByte(c byte) bool {
	return c == '/' || c == '\\'
}

// matchGlobPrefix matches the leading segments of p against the segments of
// pattern with filepath.Match; deeper segments of p are allowed.
func matchGlobPrefix(p, pattern string) bool {
	for {
		patSeg, patRest, patMore := strings.Cut(pattern, "/")
		seg, rest, more := strings.Cut(p, "/")
		if matched, err := filepath.Match(patSeg, seg); err != nil || !matched {
			return false
		}
		if !patMore {
			return true
		}
		if !more {
			return false
		}
		pattern, p = patRest, rest
	}
}

// getDefaultDeniedPaths returns default paths that should be denied access
func getDefaultDeniedPaths() []string {
	paths := []string{
		// Unix/Linux critical paths
		"/etc", "/proc", "/sys", "/dev", "/root", "/boot",
		"/usr/bin", "/usr/sbin", "/sbin", "/bin",

		// Common sensitive directories
		"/home/*/.ssh", "/home/*/.gnupg", "/var/log",

		// Windows critical paths
		"C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)",
		"C:\\Users\\Default", "C:\\ProgramData",
	}

	return paths
}

// =============================================================================
// FILE PERMISSION ANALYSIS METHODS
// =============================================================================

// analyzeBasicPermissions analyzes basic read/write/execute permissions
func (fp *FilePermission) analyzeBasicPermissions() {
	mode := fp.Mode

	if fp.IsDirectory {
		// Directory permissions
		fp.IsReadable = mode&0400 != 0   // Owner read
		fp.IsWritable = mode&0200 != 0   // Owner write
		fp.IsExecutable = mode&0100 != 0 // Owner execute (enter directory)
	} else {
		// File permissions
		fp.IsReadable = mode&0400 != 0   // Owner read
		fp.IsWritable = mode&0200 != 0   // Owner write
		fp.IsExecutable = mode&0100 != 0 // Owner execute
	}

	// Check if file is read-only (no write permission)
	fp.IsReadOnly = !fp.IsWritable
}

// analyzePlatformSpecific performs platform-specific permission analysis
func (fp *FilePermission) analyzePlatformSpecific() {
	// On Windows, use different logic for read-only detection
	if runtime.GOOS == "windows" {
		fp.analyzeWindowsPermissions()
	} else {
		fp.analyzeUnixPermissions()
	}
}

// analyzeWindowsPermissions handles Windows-specific permission analysis
func (fp *FilePermission) analyzeWindowsPermissions() {
	// Windows read-only attribute
	if fp.Mode&os.ModeType == 0 { // Regular file
		// On Windows, read-only is indicated differently
		// This is a simplified check; full implementation would use Windows API
		fp.Owner = "N/A (Windows)"
		fp.Group = "N/A (Windows)"
	}
}

// analyzeUnixPermissions handles Unix/Linux-specific permission analysis
func (fp *FilePermission) analyzeUnixPermissions() {
	// Extract permission bits more granularly
	mode := fp.Mode

	// Owner permissions
	ownerRead := mode&0400 != 0
	ownerWrite := mode&0200 != 0
	ownerExecute := mode&0100 != 0

	// Group permissions
	groupWrite := mode&0020 != 0

	// Other permissions
	otherWrite := mode&0002 != 0

	fp.IsReadable = ownerRead
	fp.IsWritable = ownerWrite
	fp.IsExecutable = ownerExecute

	// WHY: the owner bits describe the owner, not the user running the
	// program; ask the kernel what the current user may actually do.
	fp.checkCurrentUserAccess()

	fp.Owner = "owner" // Placeholder - would use syscalls for actual owner
	fp.Group = "group" // Placeholder - would use syscalls for actual group

	// Update read-only status
	fp.IsReadOnly = !ownerWrite && !groupWrite && !otherWrite
}

// assessSecurityRisk evaluates the security risk of the file permissions
func (fp *FilePermission) assessSecurityRisk() {
	mode := fp.Mode

	// Start with low risk
	fp.SecurityRisk = "low"

	// Check for high-risk permissions
	if mode&0002 != 0 { // World writable
		fp.SecurityRisk = "critical"
		return
	}

	if mode&0020 != 0 { // Group writable
		fp.SecurityRisk = "medium"
	}

	if fp.IsExecutable && fp.IsWritable {
		fp.SecurityRisk = "high" // Writable executable
	}

	// Setuid/setgid files are high risk
	if mode&os.ModeSetuid != 0 || mode&os.ModeSetgid != 0 {
		fp.SecurityRisk = "high"
	}

	// Special files and devices
	if mode&os.ModeDevice != 0 || mode&os.ModeCharDevice != 0 {
		fp.SecurityRisk = "high"
	}
}

// generateRecommendations creates security recommendations based on analysis
func (fp *FilePermission) generateRecommendations() {
	switch fp.SecurityRisk {
	case "critical":
		fp.Recommendation = "URGENT: Remove world-writable permissions. Use chmod o-w."
	case "high":
		if fp.IsExecutable && fp.IsWritable {
			fp.Recommendation = "Consider removing write permission from executable files."
		} else {
			fp.Recommendation = "Review file permissions and restrict access if possible."
		}
	case "medium":
		fp.Recommendation = "Consider restricting group write access if not needed."
	default:
		fp.Recommendation = "Permissions appear secure."
	}

	// Additional recommendations for read-only files
	if fp.IsReadOnly {
		fp.Recommendation += " File is read-only, which is good for security."
	}
}

// =============================================================================
// UTILITY METHODS
// =============================================================================

// addError adds an error to the path security result and marks it as invalid
func (result *PathSecurityResult) addError(message string) {
	result.Errors = append(result.Errors, message)
	result.IsValid = false
}

// String returns a human-readable representation of file permissions
func (fp *FilePermission) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Path: %s", fp.Path))
	parts = append(parts, fmt.Sprintf("Mode: %s", fp.Mode))

	if fp.IsReadOnly {
		parts = append(parts, "Status: READ-ONLY")
	} else {
		parts = append(parts, "Status: Writable")
	}

	parts = append(parts, fmt.Sprintf("Risk: %s", fp.SecurityRisk))

	if fp.Recommendation != "" {
		parts = append(parts, fmt.Sprintf("Recommendation: %s", fp.Recommendation))
	}

	return strings.Join(parts, "\n")
}

// IsSecure returns true if the path passes all security validations
func (result *PathSecurityResult) IsSecure() bool {
	return result.IsValid && len(result.Errors) == 0
}

// GetRiskLevel returns the numeric risk level (0=low, 1=medium, 2=high, 3=critical)
func (result *PathSecurityResult) GetRiskLevel() int {
	switch result.Risk {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	case "critical":
		return 3
	default:
		return 0
	}
}
