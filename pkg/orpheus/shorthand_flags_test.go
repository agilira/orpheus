// shorthand_flags_test.go: a shorthand accepted by the API must reach the parser.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"reflect"
	"testing"
)

func TestAddFloat64FlagShorthandWorks(t *testing.T) {
	var rate float64

	cmd := NewCommand("sample", "sample it").
		AddFloat64Flag("rate", "r", 1.0, "sampling rate").
		SetHandler(func(ctx *Context) error {
			rate = ctx.GetFlagFloat64("rate")
			return nil
		})

	app := New("myapp")
	app.AddCommand(cmd)

	if err := app.Run([]string{"sample", "-r", "0.25"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if rate != 0.25 {
		t.Fatalf("rate = %v, want 0.25: the shorthand was dropped", rate)
	}
}

func TestAddStringSliceFlagShorthandWorks(t *testing.T) {
	var tags []string

	cmd := NewCommand("deploy", "deploy it").
		AddStringSliceFlag("tags", "t", []string{"default"}, "service tags").
		SetHandler(func(ctx *Context) error {
			tags = ctx.GetFlagStringSlice("tags")
			return nil
		})

	app := New("myapp")
	app.AddCommand(cmd)

	if err := app.Run([]string{"deploy", "-t", "web,api"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"web", "api"}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("tags = %q, want %q: the shorthand was dropped", tags, want)
	}
}

// The long form keeps working when no shorthand is given.
func TestAddFlagsWithoutShorthandStillParse(t *testing.T) {
	var rate float64
	var tags []string

	cmd := NewCommand("run", "run it").
		AddFloat64Flag("rate", "", 1.0, "sampling rate").
		AddStringSliceFlag("tags", "", nil, "service tags").
		SetHandler(func(ctx *Context) error {
			rate = ctx.GetFlagFloat64("rate")
			tags = ctx.GetFlagStringSlice("tags")
			return nil
		})

	app := New("myapp")
	app.AddCommand(cmd)

	if err := app.Run([]string{"run", "--rate=0.5", "--tags=a,b"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if rate != 0.5 || !reflect.DeepEqual(tags, []string{"a", "b"}) {
		t.Fatalf("rate = %v, tags = %q", rate, tags)
	}
}
