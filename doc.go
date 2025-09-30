// Package orpheus provides an ultra-fast, professionalgrade CLI framework built on flash-flags
// with comprehensive security validation and Red Team tested security controls.
//
// Orpheus is designed to be a lightweight, high-performance CLI framework
// with zero external dependencies, professional-grade security, and a focus on simplicity.
//
// Key Features:
//   - 7x-53x faster than alternatives thanks to flash-flags integration
//   - Zero external dependencies
//   - Enterprise-grade security with comprehensive input validation
//   - Red Team tested security controls (142+ test cases)
//   - Built-in protection against path traversal, injection attacks, and malicious input
//   - Comprehensive file permission analysis and system security
//   - Thread-safe concurrent operations with race condition protection
//   - Memory leak prevention and cache management
//   - Performance-optimized security validation (~3.7μs path validation)
//   - Simple, intuitive API for rapid development
//   - Built-in auto-completion support
//   - Memory-efficient command dispatch
//
// Security Features:
//   - Path traversal attack prevention (../../../etc/passwd protection)
//   - Command injection blocking (shell metacharacter filtering)
//   - SQL injection prevention with pattern detection
//   - Environment variable validation and sanitization
//   - File system security controls and permission analysis
//   - Windows-specific attack prevention (ADS, device names)
//   - Unicode normalization attack protection
//   - Null byte injection prevention
//   - Buffer overflow protection with length limits
//   - Concurrency safety and race condition prevention
//   - Cache poisoning protection with secure eviction
//
// Basic Usage:
//
//	app := orpheus.New("myapp")
//	app.Command("install", "Install a package", func(ctx *orpheus.Context) error {
//		// All input validation is automatically handled
//		// Installation logic with enterprise-grade security
//		return nil
//	})
//	app.Run(os.Args[1:])
//
// Security Validation:
//
//	// Automatic path validation with security controls
//	if err := validator.ValidatePathFlag("config.yml", "read"); err != nil {
//		// Path traversal or malicious path detected
//		return err
//	}
//
//	// String input validation with injection prevention
//	if err := validator.ValidateStringFlag(userInput); err != nil {
//		// Command injection or malicious input detected
//		return err
//	}
//
// For more examples, advanced usage, and security guidelines, see:
//   - examples/filemanager directory
//   - docs/SECURITY.md for comprehensive security documentation
//   - docs/API.md for complete API reference
package orpheus
