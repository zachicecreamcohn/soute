//go:build windows

package ipc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os/user"
	"time"

	"github.com/Microsoft/go-winio"
)

// EndpointFor returns a named pipe address unique to the target's absolute
// path. Named pipes are machine-global (unlike UDS, which is scoped to the
// namespace directory), so the path hash disambiguates concurrent daemons.
func EndpointFor(namespaceDir, targetAbs string) string {
	sum := sha256.Sum256([]byte(targetAbs))
	return `\\.\pipe\soute-` + hex.EncodeToString(sum[:8])
}

// Listen opens a named pipe restricted to the current user via an SDDL ACL.
func Listen(addr string) (net.Listener, error) {
	sddl, err := currentUserSDDL()
	if err != nil {
		return nil, err
	}
	return winio.ListenPipe(addr, &winio.PipeConfig{SecurityDescriptor: sddl})
}

// Dial connects to the named pipe at addr.
func Dial(ctx context.Context, addr string) (net.Conn, error) {
	timeout := dialTimeout
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.Until(dl)
	}
	return winio.DialPipe(addr, &timeout)
}

// Cleanup is a no-op: named pipes leave no filesystem artifact.
func Cleanup(addr string) {}

// currentUserSDDL builds an SDDL string granting the current user full access
// only. On Windows, os/user.Current().Uid returns the user's SID.
func currentUserSDDL() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("current user: %w", err)
	}
	return "D:P(A;;GA;;;" + u.Uid + ")", nil
}
