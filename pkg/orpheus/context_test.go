// context_test.go: Orpheus application framework context tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func TestContextArguments(t *testing.T) {
	ctx := &orpheus.Context{
		Args: []string{"arg1", "arg2", "arg3"},
	}

	if ctx.ArgCount() != 3 {
		t.Errorf("expected 3 args, got %d", ctx.ArgCount())
	}

	if ctx.GetArg(0) != "arg1" {
		t.Errorf("expected arg[0] = 'arg1', got '%s'", ctx.GetArg(0))
	}

	if ctx.GetArg(1) != "arg2" {
		t.Errorf("expected arg[1] = 'arg2', got '%s'", ctx.GetArg(1))
	}

	if ctx.GetArg(2) != "arg3" {
		t.Errorf("expected arg[2] = 'arg3', got '%s'", ctx.GetArg(2))
	}
}

func TestContextOutOfBoundsArgs(t *testing.T) {
	ctx := &orpheus.Context{
		Args: []string{"arg1"},
	}

	// Test negative index
	if ctx.GetArg(-1) != "" {
		t.Errorf("expected empty string for negative index, got '%s'", ctx.GetArg(-1))
	}

	// Test index beyond bounds
	if ctx.GetArg(5) != "" {
		t.Errorf("expected empty string for out of bounds index, got '%s'", ctx.GetArg(5))
	}

	// Test exactly at bounds
	if ctx.GetArg(1) != "" {
		t.Errorf("expected empty string for index at bounds, got '%s'", ctx.GetArg(1))
	}
}

func TestContextEmptyArgs(t *testing.T) {
	ctx := &orpheus.Context{
		Args: []string{},
	}

	if ctx.ArgCount() != 0 {
		t.Errorf("expected 0 args, got %d", ctx.ArgCount())
	}

	if ctx.GetArg(0) != "" {
		t.Errorf("expected empty string for empty args, got '%s'", ctx.GetArg(0))
	}
}

func TestContextNilArgs(t *testing.T) {
	ctx := &orpheus.Context{
		Args: nil,
	}

	if ctx.ArgCount() != 0 {
		t.Errorf("expected 0 args for nil slice, got %d", ctx.ArgCount())
	}

	if ctx.GetArg(0) != "" {
		t.Errorf("expected empty string for nil args, got '%s'", ctx.GetArg(0))
	}
}

func TestContextFlags(t *testing.T) {
	ctx := &orpheus.Context{}

	// Test flag methods return default values when no flags are set
	if ctx.GetFlag("test") != nil {
		t.Errorf("expected nil for non-existent flag, got %v", ctx.GetFlag("test"))
	}

	if ctx.GetFlagString("test") != "" {
		t.Errorf("expected empty string for non-existent string flag, got '%s'", ctx.GetFlagString("test"))
	}

	if ctx.GetFlagBool("test") != false {
		t.Errorf("expected false for non-existent bool flag, got %v", ctx.GetFlagBool("test"))
	}

	if ctx.GetFlagInt("test") != 0 {
		t.Errorf("expected 0 for non-existent int flag, got %d", ctx.GetFlagInt("test"))
	}
}

func TestContextGlobalFlags(t *testing.T) {
	ctx := &orpheus.Context{}

	// Test global flag methods return default values when no flags are set
	if ctx.GetGlobalFlag("test") != nil {
		t.Errorf("expected nil for non-existent global flag, got %v", ctx.GetGlobalFlag("test"))
	}

	if ctx.GetGlobalFlagBool("test") != false {
		t.Errorf("expected false for non-existent global bool flag, got %v", ctx.GetGlobalFlagBool("test"))
	}

	// Test additional global flag methods for non-existent flags
	if ctx.GetGlobalFlagString("test") != "" {
		t.Errorf("expected empty string for non-existent global string flag, got %v", ctx.GetGlobalFlagString("test"))
	}

	if ctx.GetGlobalFlagInt("test") != 0 {
		t.Errorf("expected 0 for non-existent global int flag, got %v", ctx.GetGlobalFlagInt("test"))
	}

	if ctx.GlobalFlagChanged("test") != false {
		t.Errorf("expected false for non-existent global flag changed, got %v", ctx.GlobalFlagChanged("test"))
	}
}

// TestContextObservabilityGetters tests context observability method getters
func TestContextObservabilityGetters(t *testing.T) {
	// Test with nil app
	ctx := &orpheus.Context{App: nil}

	if ctx.Logger() != nil {
		t.Error("expected nil logger when app is nil")
	}

	if ctx.AuditLogger() != nil {
		t.Error("expected nil audit logger when app is nil")
	}

	if ctx.Tracer() != nil {
		t.Error("expected nil tracer when app is nil")
	}

	if ctx.MetricsCollector() != nil {
		t.Error("expected nil metrics collector when app is nil")
	}

	// Test with app but no observability components set
	app := orpheus.New("testapp")
	ctx = &orpheus.Context{App: app}

	if ctx.Logger() != nil {
		t.Error("expected nil logger when not configured in app")
	}

	if ctx.AuditLogger() != nil {
		t.Error("expected nil audit logger when not configured in app")
	}

	if ctx.Tracer() != nil {
		t.Error("expected nil tracer when not configured in app")
	}

	if ctx.MetricsCollector() != nil {
		t.Error("expected nil metrics collector when not configured in app")
	}
}

// TestContextFlagChangedEdgeCases tests edge cases for FlagChanged method
func TestContextFlagChangedEdgeCases(t *testing.T) {
	// Test with nil flags
	ctx := &orpheus.Context{
		Args:  []string{"test"},
		Flags: nil,
	}

	// Test FlagChanged with nil flags - should not panic and return false
	if ctx.FlagChanged("debug") {
		t.Error("FlagChanged should return false with nil flags")
	}

	if ctx.FlagChanged("nonexistent") {
		t.Error("FlagChanged should return false for nonexistent flag with nil flags")
	}

	// Test edge case: empty flag name
	if ctx.FlagChanged("") {
		t.Error("FlagChanged should return false for empty flag name")
	}
}

func TestContextGetFlagFloat64AndStringSlice(t *testing.T) {
	// Test with nil flags (edge case coverage)
	ctx := &orpheus.Context{
		Args:  []string{"test"},
		Flags: nil,
	}

	// Test GetFlagFloat64 with nil flags
	defaultRate := ctx.GetFlagFloat64("rate")
	if defaultRate != 0.0 {
		t.Errorf("Expected default rate 0.0 with nil flags, got %f", defaultRate)
	}

	// Test GetFlagStringSlice with nil flags
	defaultTags := ctx.GetFlagStringSlice("tags")
	if len(defaultTags) != 0 {
		t.Errorf("Expected empty slice with nil flags, got %v", defaultTags)
	}

	// Verify the methods don't panic with nil flags
	intVal := ctx.GetFlagInt("count")
	if intVal != 0 {
		t.Errorf("Expected 0 for int flag with nil flags, got %d", intVal)
	}

	flagVal := ctx.GetFlag("name")
	if flagVal != nil {
		t.Errorf("Expected nil for flag with nil flags, got %v", flagVal)
	}
}
