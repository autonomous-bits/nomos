package providercmd

import (
	"errors"
	"fmt"
	"testing"
)

// TestSentinelErrors_AreDistinct verifies each sentinel error is unique.
func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinelErrors := []error{
		ErrVersionConflict,
		ErrMissingVersion,
		ErrChecksumMismatch,
		ErrDownloadFailed,
	}

	// Verify all errors are non-nil
	for i, err := range sentinelErrors {
		if err == nil {
			t.Errorf("sentinel error at index %d is nil", i)
		}
	}

	// Verify all errors are distinct (no duplicates)
	for i, err1 := range sentinelErrors {
		for j, err2 := range sentinelErrors {
			if i != j && err1 == err2 {
				t.Errorf("sentinel errors at index %d and %d are identical: %v", i, j, err1)
			}
		}
	}
}

// TestSentinelErrors_Messages verifies error messages are descriptive.
func TestSentinelErrors_Messages(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantMsgContains string
	}{
		{
			name:            "ErrVersionConflict",
			err:             ErrVersionConflict,
			wantMsgContains: "conflicting",
		},
		{
			name:            "ErrMissingVersion",
			err:             ErrMissingVersion,
			wantMsgContains: "missing version",
		},
		{
			name:            "ErrChecksumMismatch",
			err:             ErrChecksumMismatch,
			wantMsgContains: "checksum mismatch",
		},
		{
			name:            "ErrDownloadFailed",
			err:             ErrDownloadFailed,
			wantMsgContains: "download failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg == "" {
				t.Error("error message is empty")
			}
			if !contains(msg, tt.wantMsgContains) {
				t.Errorf("error message = %q, want to contain %q", msg, tt.wantMsgContains)
			}
		})
	}
}

// TestSentinelErrors_Wrapping verifies errors.Is works with wrapped sentinel errors.
func TestSentinelErrors_Wrapping(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		wrapped  error
	}{
		{
			name:     "ErrVersionConflict wrapped",
			sentinel: ErrVersionConflict,
			wrapped:  fmt.Errorf("provider aws: %w", ErrVersionConflict),
		},
		{
			name:     "ErrMissingVersion wrapped",
			sentinel: ErrMissingVersion,
			wrapped:  fmt.Errorf("validation failed: %w", ErrMissingVersion),
		},
		{
			name:     "ErrChecksumMismatch wrapped",
			sentinel: ErrChecksumMismatch,
			wrapped:  fmt.Errorf("provider validation failed: %w", ErrChecksumMismatch),
		},
		{
			name:     "ErrDownloadFailed wrapped",
			sentinel: ErrDownloadFailed,
			wrapped:  fmt.Errorf("github error: %w", ErrDownloadFailed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.wrapped, tt.sentinel)
			}
		})
	}
}

// TestSentinelErrors_NotEqual verifies sentinel errors don't match other errors.
func TestSentinelErrors_NotEqual(t *testing.T) {
	differentError := errors.New("some other error")

	sentinels := []error{
		ErrVersionConflict,
		ErrMissingVersion,
		ErrChecksumMismatch,
		ErrDownloadFailed,
	}

	for _, sentinel := range sentinels {
		if errors.Is(differentError, sentinel) {
			t.Errorf("errors.Is(%v, %v) = true, want false", differentError, sentinel)
		}
		if errors.Is(sentinel, differentError) {
			t.Errorf("errors.Is(%v, %v) = true, want false", sentinel, differentError)
		}
	}
}

// TestSentinelErrors_NestedWrapping verifies deeply nested error wrapping.
func TestSentinelErrors_NestedWrapping(t *testing.T) {
	// Create a deeply nested error chain
	err1 := fmt.Errorf("context: %w", ErrChecksumMismatch)
	err2 := fmt.Errorf("layer 2: %w", err1)
	err3 := fmt.Errorf("layer 3: %w", err2)

	if !errors.Is(err3, ErrChecksumMismatch) {
		t.Errorf("errors.Is with deeply nested wrapping failed")
	}
}
