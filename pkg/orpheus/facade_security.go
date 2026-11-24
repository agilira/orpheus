// facade_security.go: the security API under its v1.11.0 names
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import "github.com/agilira/orpheus/security"

// Security and validation types, see package security.
type (
	SecurityConfig     = security.Config
	FilePermission     = security.FilePermission
	PathSecurityResult = security.PathSecurityResult
	ValidationConfig   = security.ValidationConfig
	ValidatedInput     = security.ValidatedInput
	InputValidator     = security.InputValidator
)

// DefaultSecurityConfig calls security.DefaultConfig.
func DefaultSecurityConfig() SecurityConfig { return security.DefaultConfig() }

// ValidateSecurePath calls security.ValidateSecurePath.
func ValidateSecurePath(path string, config SecurityConfig) *PathSecurityResult {
	return security.ValidateSecurePath(path, config)
}

// AnalyzeFilePermissions calls security.AnalyzeFilePermissions.
func AnalyzeFilePermissions(path string, config SecurityConfig) (*FilePermission, error) {
	return security.AnalyzeFilePermissions(path, config)
}

// CheckFileReadable calls security.CheckFileReadable.
func CheckFileReadable(path string, config SecurityConfig) (bool, error) {
	return security.CheckFileReadable(path, config)
}

// CheckFileWritable calls security.CheckFileWritable.
func CheckFileWritable(path string, config SecurityConfig) (bool, error) {
	return security.CheckFileWritable(path, config)
}

// CheckFileExecutable calls security.CheckFileExecutable.
func CheckFileExecutable(path string, config SecurityConfig) (bool, error) {
	return security.CheckFileExecutable(path, config)
}

// DefaultValidationConfig calls security.DefaultValidationConfig.
func DefaultValidationConfig() ValidationConfig { return security.DefaultValidationConfig() }

// NewInputValidator calls security.NewInputValidator.
func NewInputValidator(config ValidationConfig) *InputValidator {
	return security.NewInputValidator(config)
}
