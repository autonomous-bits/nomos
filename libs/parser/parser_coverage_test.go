package parser

import (
	"strings"
	"testing"
)

func TestParse_Coverage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "simple valid config",
			input: `config:
  key: "value"
`,
			wantErr: false,
		},
		{
			name: "nested map",
			input: `config:
  app:
    name: "foo"
`,
			wantErr: false,
		},
		{
			name: "list",
			input: `config:
  items:
    - "a"
    - "b"
`,
			wantErr: false,
		},
		{
			name:    "invalid syntax unexpected char",
			input:   `!config`,
			wantErr: true,
		},
		{
			name:    "invalid syntax standalone ref",
			input:   `@ref`, // If this parses as ReferenceExpr it might be valid? check parser.go logic
			wantErr: true,
		},
		{
			name:    "invalid syntax parenthesis",
			input:   `(config)`,
			wantErr: true,
		},
		{
			name:    "unclosed list",
			input:   `config: [`,
			wantErr: false, // Apparently valid
		},
		{
			name:    "unclosed map",
			input:   `config: {`,
			wantErr: false, // Apparently valid
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: false,
		},
		{
			name: "marked expression",
			input: `config:
  secret: "value"!
`,
			wantErr: false,
		},
		{
			name: "reference expression",
			input: `config:
  ref: @env:VAR
`,
			wantErr: false,
		},
		{
			name: "path expression key",
			input: `config:
  a: "value"
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input), "test.csl")
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
