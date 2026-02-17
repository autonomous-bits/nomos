package models

// Secret wraps a value that should be encrypted in the final output.
type Secret struct {
	Value any `json:"value"`
}
