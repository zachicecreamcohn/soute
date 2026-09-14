//go:build darwin || linux

package ipc

import (
	"context"
	"net"
	"os"
	"path/filepath"
)

const sockFileName = "soute.sock"

// EndpointFor returns the Unix domain socket path inside the namespace dir.
// Security relies on the namespace directory's 0700 permissions.
func EndpointFor(namespaceDir, targetAbs string) string {
	return filepath.Join(namespaceDir, sockFileName)
}

// Listen opens a Unix domain socket listener at addr.
func Listen(addr string) (net.Listener, error) {
	return net.Listen("unix", addr)
}

// Dial connects to the Unix domain socket at addr.
func Dial(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: dialTimeout}
	return d.DialContext(ctx, "unix", addr)
}

// Cleanup removes a stale socket file, ignoring absence.
func Cleanup(addr string) {
	_ = os.Remove(addr)
}
