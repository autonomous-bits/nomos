package serialize

import (
	"strings"
	"testing"
)

// TestOutputFormat_Validate tests validation of OutputFormat enum values.
func TestOutputFormat_Validate(t *testing.T) {
	tests := []struct {
		name    string
		format  OutputFormat
		wantErr bool
		errMsg  string
	}{
		{
			name:    "json format valid",
			format:  FormatJSON,
			wantErr: false,
		},
		{
			name:    "yaml format valid",
			format:  FormatYAML,
			wantErr: false,
		},
		{
			name:    "tfvars format valid",
			format:  FormatTfvars,
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  OutputFormat("invalid"),
			wantErr: true,
			errMsg:  "unsupported format",
		},
		{
			name:    "empty format",
			format:  OutputFormat(""),
			wantErr: true,
			errMsg:  "unsupported format",
		},
		{
			name:    "uppercase JSON rejected",
			format:  OutputFormat("JSON"),
			wantErr: true,
			errMsg:  "unsupported format",
		},
		{
			name:    "mixed case YAML rejected",
			format:  OutputFormat("Yaml"),
			wantErr: true,
			errMsg:  "unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.format.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want substring %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOutputFormat_Extension tests file extension mapping for each format.
func TestOutputFormat_Extension(t *testing.T) {
	tests := []struct {
		name      string
		format    OutputFormat
		wantExt   string
		mustStart string // Extension must start with this
	}{
		{
			name:      "json extension",
			format:    FormatJSON,
			wantExt:   ".json",
			mustStart: ".",
		},
		{
			name:      "yaml extension",
			format:    FormatYAML,
			wantExt:   ".yaml",
			mustStart: ".",
		},
		{
			name:      "tfvars extension",
			format:    FormatTfvars,
			wantExt:   ".tfvars",
			mustStart: ".",
		},
		{
			name:    "invalid format returns empty",
			format:  OutputFormat("invalid"),
			wantExt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.format.Extension()
			if got != tt.wantExt {
				t.Errorf("Extension() = %q, want %q", got, tt.wantExt)
			}

			if tt.mustStart != "" && !strings.HasPrefix(got, tt.mustStart) {
				t.Errorf("Extension() = %q, must start with %q", got, tt.mustStart)
			}
		})
	}
}

// TestOutputFormat_CaseNormalization tests that format comparison is case-sensitive.
// This ensures callers must normalize to lowercase before validation.
func TestOutputFormat_CaseNormalization(t *testing.T) {
	// These should all fail validation (case-sensitive comparison)
	invalidCases := []OutputFormat{
		"JSON",
		"Json",
		"YAML",
		"Yaml",
		"TFVARS",
		"Tfvars",
		"TfVars",
	}

	for _, format := range invalidCases {
		t.Run(string(format), func(t *testing.T) {
			err := format.Validate()
			if err == nil {
				t.Errorf("Validate() expected error for %q (case-sensitive)", format)
			}
		})
	}
}

// TestOutputFormat_ErrorMessage tests that error messages list supported formats.
func TestOutputFormat_ErrorMessage(t *testing.T) {
	format := OutputFormat("xml")
	err := format.Validate()

	if err == nil {
		t.Fatal("Validate() expected error for unsupported format")
	}

	errMsg := err.Error()

	// Error message must include all supported formats
	requiredSubstrings := []string{
		"unsupported format",
		"json",
		"yaml",
		"tfvars",
	}

	for _, substring := range requiredSubstrings {
		if !strings.Contains(errMsg, substring) {
			t.Errorf("error message missing %q: %s", substring, errMsg)
		}
	}
}

// TestOutputFormat_String tests string representation of format.
func TestOutputFormat_String(t *testing.T) {
	tests := []struct {
		format OutputFormat
		want   string
	}{
		{FormatJSON, "json"},
		{FormatYAML, "yaml"},
		{FormatTfvars, "tfvars"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.format)
			if got != tt.want {
				t.Errorf("string(format) = %q, want %q", got, tt.want)
			}
		})
	}
}
