// facade_prompt.go: the prompt API under its v1.11.0 names
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"io"

	"github.com/agilira/orpheus/prompt"
)

// Prompt types, see package prompt.
type (
	Prompter         = prompt.Prompter
	TerminalPrompter = prompt.Terminal
)

// NewTerminalPrompter calls prompt.NewTerminal.
func NewTerminalPrompter() *TerminalPrompter { return prompt.NewTerminal() }

// NewPrompterFrom calls prompt.NewFrom.
func NewPrompterFrom(r io.Reader, w io.Writer, fd int) *TerminalPrompter {
	return prompt.NewFrom(r, w, fd)
}
