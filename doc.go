// Package orpheus provides an ultra-fast CLI framework built on flash-flags.
//
// Orpheus is designed to be a lightweight, high-performance cli framework
// with zero external dependencies (except flash-flags) and a focus on simplicity.
//
// Key Features:
//   - 7-53x faster than alternatives thanks to flash-flags integration
//   - Zero external dependencies beyond flash-flags
//   - Simple, intuitive API for rapid development
//   - Built-in auto-completion support
//   - Memory-efficient command dispatch
//
// Basic Usage:
//
//	app := orpheus.New("myapp")
//	app.Command("install", "Install a package", func(ctx *orpheus.Context) error {
//		// Installation logic
//		return nil
//	})
//	app.Run(os.Args[1:])
//
// For more examples and advanced usage, see the examples/filemanager directory.
package orpheus
