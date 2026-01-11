// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import "errors"

// Sentinel errors for provider management operations.
var (
	// ErrVersionConflict is returned when multiple .csl files declare the same
	// provider with different versions.
	ErrVersionConflict = errors.New("conflicting provider versions across files")

	// ErrMissingVersion is returned when a provider source declaration is
	// missing the required version field.
	ErrMissingVersion = errors.New("provider declaration missing version field")

	// ErrChecksumMismatch is returned when a downloaded provider binary's
	// SHA256 checksum doesn't match the expected value from GitHub releases.
	ErrChecksumMismatch = errors.New("downloaded binary checksum mismatch")

	// ErrDownloadFailed is returned when downloading a provider binary from
	// GitHub releases fails due to network errors, missing assets, or
	// unavailable releases.
	ErrDownloadFailed = errors.New("provider download failed")
)
