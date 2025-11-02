package downloader

import (
	"errors"
	"testing"
)

// TestErrors_Unwrap tests that typed errors properly unwrap to sentinel errors.
func TestErrors_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
	}{
		{
			name: "AssetNotFoundError unwraps to ErrAssetNotFound",
			err: &AssetNotFoundError{
				Owner:   "test-owner",
				Repo:    "test-repo",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			sentinel: ErrAssetNotFound,
		},
		{
			name: "ChecksumMismatchError unwraps to ErrChecksumMismatch",
			err: &ChecksumMismatchError{
				Expected: "abc123",
				Actual:   "def456",
			},
			sentinel: ErrChecksumMismatch,
		},
		{
			name: "InvalidSpecError unwraps to ErrInvalidSpec",
			err: &InvalidSpecError{
				Field:   "Owner",
				Message: "owner is required",
			},
			sentinel: ErrInvalidSpec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.sentinel) {
				t.Errorf("expected error to unwrap to %v", tt.sentinel)
			}
		})
	}
}

// TestAssetNotFoundError_Message tests error message formatting.
func TestAssetNotFoundError_Message(t *testing.T) {
	err := &AssetNotFoundError{
		Owner:   "autonomous-bits",
		Repo:    "nomos-provider-file",
		Version: "1.0.0",
		OS:      "linux",
		Arch:    "amd64",
	}

	expected := "asset not found for autonomous-bits/nomos-provider-file@1.0.0 (os=linux, arch=amd64)"
	if err.Error() != expected {
		t.Errorf("expected message %q, got %q", expected, err.Error())
	}
}

// TestChecksumMismatchError_Message tests error message formatting.
func TestChecksumMismatchError_Message(t *testing.T) {
	err := &ChecksumMismatchError{
		Expected: "abc123",
		Actual:   "def456",
	}

	expected := "checksum mismatch: expected=abc123, actual=def456"
	if err.Error() != expected {
		t.Errorf("expected message %q, got %q", expected, err.Error())
	}
}

// TestInvalidSpecError_Message tests error message formatting.
func TestInvalidSpecError_Message(t *testing.T) {
	err := &InvalidSpecError{
		Field:   "Owner",
		Message: "owner is required",
	}

	expected := "invalid spec field \"Owner\": owner is required"
	if err.Error() != expected {
		t.Errorf("expected message %q, got %q", expected, err.Error())
	}
}

// TestSentinelErrors_Distinct tests that sentinel errors are distinct.
func TestSentinelErrors_Distinct(t *testing.T) {
	sentinels := []error{
		ErrAssetNotFound,
		ErrChecksumMismatch,
		ErrInvalidSpec,
		ErrRateLimitExceeded,
		ErrNetworkFailure,
		ErrNotImplemented,
	}

	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("sentinel errors should not be equal: %v == %v", err1, err2)
			}
		}
	}
}
