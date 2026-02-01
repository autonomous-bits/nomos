package serialize

import "fmt"

// OutputFormat represents supported serialization formats.
type OutputFormat string

const (
	// FormatJSON is the JSON output format (default).
	FormatJSON OutputFormat = "json"

	// FormatYAML is the YAML output format.
	FormatYAML OutputFormat = "yaml"

	// FormatTfvars is the HCL .tfvars output format.
	FormatTfvars OutputFormat = "tfvars"
)

// Validate checks if the format is supported.
// Returns an error if the format is not one of: json, yaml, tfvars.
//
// Note: Validation is case-sensitive. Use strings.ToLower() before
// calling Validate() if case-insensitive format selection is needed.
func (f OutputFormat) Validate() error {
	switch f {
	case FormatJSON, FormatYAML, FormatTfvars:
		return nil
	default:
		return fmt.Errorf("unsupported format: %q (supported: json, yaml, tfvars)", f)
	}
}

// Extension returns the default file extension for this format.
// Returns:
//   - ".json" for FormatJSON
//   - ".yaml" for FormatYAML
//   - ".tfvars" for FormatTfvars
//   - "" (empty string) for invalid formats
func (f OutputFormat) Extension() string {
	switch f {
	case FormatJSON:
		return ".json"
	case FormatYAML:
		return ".yaml"
	case FormatTfvars:
		return ".tfvars"
	default:
		return ""
	}
}
