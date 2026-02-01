module github.com/autonomous-bits/nomos/libs/compiler

go 1.25.6

require (
	github.com/autonomous-bits/nomos/libs/parser v0.0.0-00010101000000-000000000000
	github.com/autonomous-bits/nomos/libs/provider-proto v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)

replace github.com/autonomous-bits/nomos/libs/parser => ../parser

replace github.com/autonomous-bits/nomos/libs/provider-proto => ../provider-proto
