// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import "fmt"

// ProviderSummary provides aggregate statistics for provider operations.
// It summarizes the results of discovering, downloading, and caching providers
// across a build operation.
type ProviderSummary struct {
	// Total is the total number of providers processed
	Total int

	// Cached is the count of providers that were already installed and reused
	Cached int

	// Downloaded is the count of providers that were newly downloaded
	Downloaded int

	// Failed is the count of providers that failed to download or install
	Failed int
}

// String returns a human-readable summary of provider operations.
// Format: "Providers: N total, N cached, N downloaded, N failed"
//
// Example: "Providers: 3 total, 2 cached, 1 downloaded, 0 failed"
func (s ProviderSummary) String() string {
	return fmt.Sprintf("Providers: %d total, %d cached, %d downloaded, %d failed",
		s.Total, s.Cached, s.Downloaded, s.Failed)
}
