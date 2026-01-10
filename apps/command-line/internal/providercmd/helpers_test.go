package providercmd

import (
	"os"
	"testing"
	"time"
)

// TestParseOwnerRepo tests the owner/repo parsing helper.
func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		name        string
		providerType string
		wantOwner   string
		wantRepo    string
		wantErr     bool
		errContains string
	}{
		{
			name:         "valid owner/repo",
			providerType: "autonomous-bits/nomos-provider-file",
			wantOwner:    "autonomous-bits",
			wantRepo:     "nomos-provider-file",
			wantErr:      false,
		},
		{
			name:         "simple owner/repo",
			providerType: "owner/repo",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantErr:      false,
		},
		{
			name:         "owner with hyphen",
			providerType: "my-org/my-repo",
			wantOwner:    "my-org",
			wantRepo:     "my-repo",
			wantErr:      false,
		},
		{
			name:         "no slash",
			providerType: "invalidtype",
			wantErr:      true,
			errContains:  "must be in 'owner/repo' format",
		},
		{
			name:         "multiple slashes",
			providerType: "owner/repo/extra",
			wantErr:      true,
			errContains:  "multiple slashes",
		},
		{
			name:         "empty owner",
			providerType: "/repo",
			wantErr:      true,
			errContains:  "non-empty owner and repo",
		},
		{
			name:         "empty repo",
			providerType: "owner/",
			wantErr:      true,
			errContains:  "non-empty owner and repo",
		},
		{
			name:         "just slash",
			providerType: "/",
			wantErr:      true,
			errContains:  "non-empty owner and repo",
		},
		{
			name:         "empty string",
			providerType: "",
			wantErr:      true,
			errContains:  "must be in 'owner/repo' format",
		},
		{
			name:         "trailing slash",
			providerType: "owner/repo/",
			wantErr:      true,
			errContains:  "multiple slashes",
		},
		{
			name:         "leading slash",
			providerType: "/owner/repo",
			wantErr:      true,
			errContains:  "multiple slashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseOwnerRepo(tt.providerType)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseOwnerRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

// TestExprToValue tests the AST expression conversion helper.
func TestExprToValue(t *testing.T) {
	tests := []struct {
		name string
		expr interface{} // We'll use interface{} to simulate ast.Expr
		want interface{}
	}{
		{
			name: "nil expr",
			expr: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Full testing of exprToValue requires proper AST types
			// This is tested indirectly through DiscoverProviders tests
			// which use real parsed AST nodes from the parser
			t.Skip("exprToValue is tested indirectly through DiscoverProviders tests")
		})
	}
}

// TestTimeNowRFC3339 tests the timestamp helper.
func TestTimeNowRFC3339(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantVal string
	}{
		{
			name:    "without test env var",
			envVar:  "",
			wantVal: "", // Will check it's a valid RFC3339 string
		},
		{
			name:    "with test env var",
			envVar:  "2026-01-10T12:00:00Z",
			wantVal: "2026-01-10T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env var
			origEnv := os.Getenv("NOMOS_TEST_TIMESTAMP")
			defer func() {
				if origEnv != "" {
					os.Setenv("NOMOS_TEST_TIMESTAMP", origEnv)
				} else {
					os.Unsetenv("NOMOS_TEST_TIMESTAMP")
				}
			}()

			// Set test env var
			if tt.envVar != "" {
				os.Setenv("NOMOS_TEST_TIMESTAMP", tt.envVar)
			} else {
				os.Unsetenv("NOMOS_TEST_TIMESTAMP")
			}

			got := timeNowRFC3339()

			if tt.wantVal != "" {
				if got != tt.wantVal {
					t.Errorf("timeNowRFC3339() = %q, want %q", got, tt.wantVal)
				}
			} else {
				// Verify it's a valid RFC3339 string
				if _, err := time.Parse(time.RFC3339, got); err != nil {
					t.Errorf("timeNowRFC3339() returned invalid RFC3339: %v", err)
				}
			}
		})
	}
}
