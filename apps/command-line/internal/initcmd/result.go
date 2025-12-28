// Package initcmd implements result types for the init command.
package initcmd

// InitResult represents the result of running the init command.
type InitResult struct {
	// Providers lists all providers that were processed
	Providers []ProviderResult

	// Skipped is the count of providers that were already installed
	Skipped int

	// Installed is the count of providers that were newly installed
	Installed int

	// DryRun indicates if this was a dry-run operation
	DryRun bool
}

// ProviderResult represents the result of processing a single provider.
type ProviderResult struct {
	// Alias is the provider alias
	Alias string

	// Type is the provider type (owner/repo format)
	Type string

	// Version is the provider version
	Version string

	// OS is the target operating system
	OS string

	// Arch is the target architecture
	Arch string

	// Status indicates what happened with this provider
	Status ProviderStatus

	// Error is set if the provider installation failed
	Error error

	// Size is the download size in bytes (0 if not applicable)
	Size int64

	// Path is the installation path
	Path string
}

// ProviderStatus represents the status of a provider installation.
type ProviderStatus string

const (
	// ProviderStatusSkipped indicates the provider was already installed
	ProviderStatusSkipped ProviderStatus = "skipped"

	// ProviderStatusInstalled indicates the provider was newly installed
	ProviderStatusInstalled ProviderStatus = "installed"

	// ProviderStatusFailed indicates the provider installation failed
	ProviderStatusFailed ProviderStatus = "failed"

	// ProviderStatusDryRun indicates this is a dry-run preview
	ProviderStatusDryRun ProviderStatus = "dry-run"
)
