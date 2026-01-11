package providercmd

import (
	"testing"
)

// TestProviderSummary_String tests the String() method formatting.
func TestProviderSummary_String(t *testing.T) {
	tests := []struct {
		name    string
		summary ProviderSummary
		want    string
	}{
		{
			name: "all zeros",
			summary: ProviderSummary{
				Total:      0,
				Cached:     0,
				Downloaded: 0,
				Failed:     0,
			},
			want: "Providers: 0 total, 0 cached, 0 downloaded, 0 failed",
		},
		{
			name: "all cached",
			summary: ProviderSummary{
				Total:      3,
				Cached:     3,
				Downloaded: 0,
				Failed:     0,
			},
			want: "Providers: 3 total, 3 cached, 0 downloaded, 0 failed",
		},
		{
			name: "all downloaded",
			summary: ProviderSummary{
				Total:      2,
				Cached:     0,
				Downloaded: 2,
				Failed:     0,
			},
			want: "Providers: 2 total, 0 cached, 2 downloaded, 0 failed",
		},
		{
			name: "mixed success",
			summary: ProviderSummary{
				Total:      5,
				Cached:     2,
				Downloaded: 3,
				Failed:     0,
			},
			want: "Providers: 5 total, 2 cached, 3 downloaded, 0 failed",
		},
		{
			name: "with failures",
			summary: ProviderSummary{
				Total:      4,
				Cached:     1,
				Downloaded: 2,
				Failed:     1,
			},
			want: "Providers: 4 total, 1 cached, 2 downloaded, 1 failed",
		},
		{
			name: "all failed",
			summary: ProviderSummary{
				Total:      3,
				Cached:     0,
				Downloaded: 0,
				Failed:     3,
			},
			want: "Providers: 3 total, 0 cached, 0 downloaded, 3 failed",
		},
		{
			name: "single provider cached",
			summary: ProviderSummary{
				Total:      1,
				Cached:     1,
				Downloaded: 0,
				Failed:     0,
			},
			want: "Providers: 1 total, 1 cached, 0 downloaded, 0 failed",
		},
		{
			name: "single provider downloaded",
			summary: ProviderSummary{
				Total:      1,
				Cached:     0,
				Downloaded: 1,
				Failed:     0,
			},
			want: "Providers: 1 total, 0 cached, 1 downloaded, 0 failed",
		},
		{
			name: "large numbers",
			summary: ProviderSummary{
				Total:      100,
				Cached:     50,
				Downloaded: 45,
				Failed:     5,
			},
			want: "Providers: 100 total, 50 cached, 45 downloaded, 5 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.String()
			if got != tt.want {
				t.Errorf("ProviderSummary.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestProviderSummary_Consistency verifies that counts add up correctly.
func TestProviderSummary_Consistency(t *testing.T) {
	tests := []struct {
		name           string
		summary        ProviderSummary
		wantConsistent bool
	}{
		{
			name: "consistent - all cached",
			summary: ProviderSummary{
				Total:      3,
				Cached:     3,
				Downloaded: 0,
				Failed:     0,
			},
			wantConsistent: true,
		},
		{
			name: "consistent - mixed",
			summary: ProviderSummary{
				Total:      5,
				Cached:     2,
				Downloaded: 2,
				Failed:     1,
			},
			wantConsistent: true,
		},
		{
			name: "inconsistent - counts exceed total",
			summary: ProviderSummary{
				Total:      3,
				Cached:     2,
				Downloaded: 2,
				Failed:     1, // 2+2+1 = 5 > 3
			},
			wantConsistent: false,
		},
		{
			name: "inconsistent - counts below total",
			summary: ProviderSummary{
				Total:      5,
				Cached:     1,
				Downloaded: 1,
				Failed:     1, // 1+1+1 = 3 < 5
			},
			wantConsistent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sumOfCounts := tt.summary.Cached + tt.summary.Downloaded + tt.summary.Failed
			isConsistent := sumOfCounts == tt.summary.Total

			if isConsistent != tt.wantConsistent {
				t.Errorf("Summary consistency = %v (sum=%d, total=%d), want %v",
					isConsistent, sumOfCounts, tt.summary.Total, tt.wantConsistent)
			}
		})
	}
}

// TestProviderSummary_ZeroValue tests behavior of zero-value summary.
func TestProviderSummary_ZeroValue(t *testing.T) {
	var summary ProviderSummary

	got := summary.String()
	want := "Providers: 0 total, 0 cached, 0 downloaded, 0 failed"

	if got != want {
		t.Errorf("zero-value String() = %q, want %q", got, want)
	}
}
