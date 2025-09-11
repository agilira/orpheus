// context_test.go: context tests
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
}
