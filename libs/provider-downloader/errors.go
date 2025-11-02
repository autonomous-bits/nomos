package downloader

import (
	"errors"
	"fmt"
)

// Common sentinel errors returned by the downloader.
var (
	// ErrAssetNotFound is returned when no matching asset is found for the given spec.
	ErrAssetNotFound = errors.New("asset not found")

	// ErrChecksumMismatch is returned when the downloaded file's checksum
	// does not match the expected checksum.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrInvalidSpec is returned when the ProviderSpec is missing required fields
	// or contains invalid values.
	ErrInvalidSpec = errors.New("invalid provider spec")

	// ErrRateLimitExceeded is returned when GitHub API rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("GitHub API rate limit exceeded")

	// ErrNetworkFailure is returned when a network error occurs during download
	// after all retry attempts have been exhausted.
	ErrNetworkFailure = errors.New("network failure")

	// ErrNotImplemented is returned for operations that are not yet implemented.
	ErrNotImplemented = errors.New("not implemented")
)

// AssetNotFoundError provides details when an asset cannot be found.
type AssetNotFoundError struct {
	Owner   string
	Repo    string
	Version string
	OS      string
	Arch    string
}

func (e *AssetNotFoundError) Error() string {
	return fmt.Sprintf("asset not found for %s/%s@%s (os=%s, arch=%s)",
		e.Owner, e.Repo, e.Version, e.OS, e.Arch)
}

func (e *AssetNotFoundError) Unwrap() error {
	return ErrAssetNotFound
}

// ChecksumMismatchError provides details when checksums don't match.
type ChecksumMismatchError struct {
	Expected string
	Actual   string
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf("checksum mismatch: expected=%s, actual=%s", e.Expected, e.Actual)
}

func (e *ChecksumMismatchError) Unwrap() error {
	return ErrChecksumMismatch
}

// InvalidSpecError provides details about invalid provider specifications.
type InvalidSpecError struct {
	Field   string
	Message string
}

func (e *InvalidSpecError) Error() string {
	return fmt.Sprintf("invalid spec field %q: %s", e.Field, e.Message)
}

func (e *InvalidSpecError) Unwrap() error {
	return ErrInvalidSpec
}
