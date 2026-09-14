package cli

import (
	"context"
	"time"

	"github.com/zacharycohn/soute/internal/ipc"
)

// ping connects to the daemon at addr and reports whether it responds.
func ping(addr string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := ipc.DialClient(ctx, addr)
	if err != nil {
		return false, err
	}
	defer c.Close()
	if _, err := c.Call(ctx, ipc.CmdPing, nil); err != nil {
		return false, err
	}
	return true, nil
}

// sendIPC dials the daemon, sends one command, and closes the connection.
func sendIPC(addr, cmd string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := ipc.DialClient(ctx, addr)
	if err != nil {
		return err
	}
	defer c.Close()
	_, err = c.Call(ctx, cmd, nil)
	return err
}
