// Package orpheus provides an ultra-fast, professional-grade CLI framework built on flash-flags
// with comprehensive security validation and Red Team tested security controls.
//
// Orpheus is designed to be a lightweight, high-performance CLI framework
// with zero external dependencies, professional-grade security, and a focus on simplicity.
//
// Key Features:
//   - Fast parsing and dispatch thanks to flash-flags: about 10x faster than
//     cobra and kingpin, 60x faster than urfave/cli, at 3 allocations per
//     dispatch (see ./benchmarks)
//   - Zero external dependencies
//   - Enterprise-grade security with comprehensive input validation
//   - Red Team tested security controls (465 test cases)
//   - Built-in protection against path traversal, injection attacks, and malicious input
//   - Comprehensive file permission analysis and system security
//   - Thread-safe concurrent operations with race condition protection
//   - Memory leak prevention and cache management
//   - Security validation measured rather than estimated: see the Benchmark
//     functions in security/bench_test.go
//   - Simple, intuitive API for rapid development
//   - Built-in auto-completion support
//   - Memory-efficient command dispatch
//   - Plugin-based extensible architecture with storage interface
//   - Dynamic plugin loading with comprehensive security validation
//   - Thread-safe storage operations with statistical tracking
//   - Interactive Prompts: Prompter interface for text, secrets, menus, confirmations
//   - Positional argument API: post-parse positionals separated from flag tokens
//   - Signal-aware Run and explicit context propagation through App.RunContext
//
// Storage Interface:
//   - Key-value storage abstraction with pluggable backends
//   - In-memory provider for ultra-fast operations (842ns SET, 116ns GET)
//   - Plugin-based architecture for custom storage implementations
//   - Full CRUD operations: Set, Get, Delete, List with prefix filtering
//   - Health monitoring and performance statistics tracking
//   - Thread-safe concurrent operations with mutex protection
//   - Data isolation guarantees and copy-on-access semantics
//   - Resource cleanup and proper connection management
//
// Security Features:
//   - Path traversal attack prevention (../../../etc/passwd protection)
//   - Command injection blocking (shell metacharacter filtering)
//   - Environment variable validation and sanitization
//   - File system security controls and permission analysis
//   - Windows-specific attack prevention (ADS, device names)
//   - Unicode normalization attack protection
//   - Null byte injection prevention
//   - Buffer overflow protection with length limits
//   - Concurrency safety and race condition prevention
//   - Cache poisoning protection with secure eviction
//   - Plugin security validation with DoS attack prevention
//   - Timeout controls for plugin discovery (2s limit, 10k files max)
//   - Symlink protection and directory traversal limits (max depth 10)
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
// Context Propagation:
//
//	app.Command("run", "Run work", func(ctx *orpheus.Context) error {
//		return doWork(ctx.Context())
//	})
//	app.Run(os.Args[1:]) // signal-aware by default
//
//	// Use RunContext when the caller owns cancellation or deadlines.
//	runCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
//	defer cancel()
//	app.RunContext(runCtx, os.Args[1:])
//
// Packages:
//
// This package holds App, Command and Context. The rest of the framework
// lives in packages under github.com/agilira/orpheus/:
//   - errs: the Error type, the ErrCode* codes and their constructors
//   - observe: Logger, AuditLogger, Tracer, MetricsCollector, Field
//   - storage: the Storage interface, Config, the plugin manager
//   - security: InputValidator, ValidateSecurePath, file permission checks
//   - prompt: the Prompter interface and its terminal implementation
//   - metrics, term: optional collectors and terminal output helpers
//
// The v1.11.0 import paths pkg/orpheus, pkg/metrics and pkg/term keep
// working: their names are aliases of the same types.
//
// Storage Usage:
//
//	// Load the storage plugin with security validation
//	app.ConfigureStorage(&storage.Config{Provider: "memory", PluginPath: "./plugins/memory.so"})
//	if err := app.StorageError(); err != nil {
//		return err
//	}
//
//	// Inside a handler
//	store, err := ctx.RequireStorage()
//	if err != nil {
//		return err
//	}
//	if err := store.Set(ctx.Context(), "key", []byte("value")); err != nil {
//		return err
//	}
//	data, err := store.Get(ctx.Context(), "key")
//	if err != nil {
//		return err
//	}
//
//	// List keys with prefix filtering
//	keys, err := store.List(ctx.Context(), "user:")
//	if err != nil {
//		return err
//	}
//
// Security Validation:
//
//	validator := security.NewInputValidator(security.DefaultValidationConfig())
//
//	// Path validation with security controls
//	if r := validator.ValidatePathFlag("config", path); !r.IsValid {
//		// Path traversal or malicious path detected
//		return errs.ValidationError("cmd", strings.Join(r.ValidationErrors, "; "))
//	}
//
//	// String input validation with injection prevention
//	if r := validator.ValidateStringFlag("name", userInput); !r.IsValid {
//		// Command injection or malicious input detected
//		return errs.ValidationError("cmd", strings.Join(r.ValidationErrors, "; "))
//	}
//
// For more examples, advanced usage, and security guidelines, see:
//   - examples/basic directory - Simple CLI application
//   - examples/storage directory - Storage plugin system with memory provider
//   - examples/filemanager directory - Advanced file operations with security
//   - docs/SECURITY.md for comprehensive security documentation
//   - docs/API.md for complete API reference (RunContext, Positional, Prompter, Storage, Errors)
//   - docs/STORAGE.md for storage interface documentation
package orpheus
