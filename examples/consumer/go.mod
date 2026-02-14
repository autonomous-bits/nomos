module github.com/autonomous-bits/nomos/examples/consumer

go 1.26.0

// This example demonstrates how external consumers should depend on Nomos libraries
// after they are published as versioned Go modules.
//
// For local development within this monorepo, the go.work file at the root
// handles module resolution automatically. External consumers would use:
//
// require (
//     github.com/autonomous-bits/nomos/libs/compiler v0.1.0
//     github.com/autonomous-bits/nomos/libs/parser v0.1.0
//     github.com/autonomous-bits/nomos/libs/provider-proto v0.1.0
// )
