package compiler

// Ensure Manager implements ProviderManager at compile time.
var _ ProviderManager = (*Manager)(nil)
