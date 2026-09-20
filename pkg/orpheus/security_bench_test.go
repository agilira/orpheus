// security_bench_test.go: benchmarks for the validation entry points whose
// doc comments quote a per-call cost.
//
// WHY they exist: those figures were written by hand and nothing measured
// them, so they drifted -- ValidateSecurePath was documented at ~200ns in
// security.go and ~3.7us in docs/SECURITY.md, an 18x disagreement about the
// same function. A number in a doc comment is a claim; this file is where it
// is checked.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkValidateSecurePath(b *testing.B) {
	config := DefaultSecurityConfig()

	b.Run("Valid", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ValidateSecurePath("data/reports/output.json", config)
		}
	})

	b.Run("Traversal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ValidateSecurePath("../../etc/passwd", config)
		}
	})
}

func BenchmarkAnalyzeFilePermissions(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		b.Fatalf("write: %v", err)
	}
	config := DefaultSecurityConfig()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = AnalyzeFilePermissions(path, config)
	}
}

func BenchmarkValidateStringFlag(b *testing.B) {
	config := DefaultValidationConfig()
	config.EnableCaching = false
	v := NewInputValidator(config)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = v.ValidateStringFlag("name", "production-cluster-01")
	}
}

func BenchmarkValidatePathFlag(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		b.Fatalf("write: %v", err)
	}

	b.Run("Uncached", func(b *testing.B) {
		config := DefaultValidationConfig()
		config.EnableCaching = false
		v := NewInputValidator(config)

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = v.ValidatePathFlag("input", path)
		}
	})

	b.Run("Cached", func(b *testing.B) {
		config := DefaultValidationConfig()
		v := NewInputValidator(config)
		_ = v.ValidatePathFlag("input", path) // warm the cache

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = v.ValidatePathFlag("input", path)
		}
	})
}

func BenchmarkValidateEnvironmentValue(b *testing.B) {
	v := NewInputValidator(DefaultValidationConfig())

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = v.ValidateEnvironmentValue("ORPHEUS_HOME", "/opt/orpheus")
	}
}

func BenchmarkValidateFileOperation(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		b.Fatalf("write: %v", err)
	}

	config := DefaultValidationConfig()
	config.EnableCaching = false
	v := NewInputValidator(config)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = v.ValidateFileOperation(path, "read")
	}
}
