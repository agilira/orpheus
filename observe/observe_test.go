// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package observe_test

import (
	"errors"
	"testing"

	"github.com/agilira/orpheus/observe"
)

func TestFieldConstructors(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		got  observe.Field
		want observe.Field
	}{
		{observe.StringField("k", "v"), observe.Field{Key: "k", Value: "v"}},
		{observe.IntField("n", 3), observe.Field{Key: "n", Value: 3}},
		{observe.Float64Field("f", 1.5), observe.Field{Key: "f", Value: 1.5}},
		{observe.BoolField("b", true), observe.Field{Key: "b", Value: true}},
		{observe.ErrorField(boom), observe.Field{Key: "error", Value: boom}},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %+v, want %+v", c.got, c.want)
		}
	}
}

func TestStatusCodes(t *testing.T) {
	if observe.StatusCodeUnset != 0 || observe.StatusCodeOK != 1 || observe.StatusCodeError != 2 {
		t.Errorf("status codes changed: %d %d %d", observe.StatusCodeUnset, observe.StatusCodeOK, observe.StatusCodeError)
	}
}
