module samuellando.com/tmixer

go 1.25.4

require (
	github.com/creack/pty v1.1.24
	github.com/goccy/go-yaml v1.19.0
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	golang.org/x/term v0.39.0
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

tool (
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
)
