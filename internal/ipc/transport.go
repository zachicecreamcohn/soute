package ipc

import "time"

// Connection and response timeouts shared by both platforms.
const (
	dialTimeout = 2 * time.Second
	readTimeout = 10 * time.Second
)

// EndpointFor, Listen, Dial, and Cleanup are implemented per-platform:
//   - transport_unix.go    (//go:build darwin || linux) — Unix domain socket
//   - transport_windows.go (//go:build windows)         — go-winio named pipe
//
// EndpointFor returns the platform-specific address for the given namespace
// directory and absolute target path. Listen opens a listener, Dial connects,
// and Cleanup idempotently removes any leftover endpoint artifact.
