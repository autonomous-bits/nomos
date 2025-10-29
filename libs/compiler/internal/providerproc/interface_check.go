package providerproc

import "github.com/autonomous-bits/nomos/libs/compiler"

// Ensure Manager implements compiler.ProviderManager at compile time.
var _ compiler.ProviderManager = (*Manager)(nil)
