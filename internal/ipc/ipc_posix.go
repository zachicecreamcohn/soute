//go:build !windows

package ipc

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type signalServer struct {
	cmds chan Command
	sig  chan os.Signal
}

// NewServer listens for SIGTERM/SIGUSR1/SIGUSR2 and maps them to commands.
func NewServer(snapDir string) (Server, error) {
	s := &signalServer{cmds: make(chan Command, 8), sig: make(chan os.Signal, 4)}
	signal.Notify(s.sig, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for sig := range s.sig {
			switch sig {
			case syscall.SIGTERM:
				s.cmds <- CmdStop
			case syscall.SIGUSR1:
				s.cmds <- CmdPause
			case syscall.SIGUSR2:
				s.cmds <- CmdResume
			}
		}
	}()
	return s, nil
}

func (s *signalServer) Commands() <-chan Command { return s.cmds }

func (s *signalServer) Close() error {
	signal.Stop(s.sig)
	return nil
}

type signalClient struct {
	snapDir string
}

func NewClient(snapDir string) Client { return &signalClient{snapDir: snapDir} }

func (c *signalClient) Send(cmd Command) error {
	pid, err := ReadPID(c.snapDir)
	if err != nil {
		return err
	}
	var sig syscall.Signal
	switch cmd {
	case CmdStop:
		sig = syscall.SIGTERM
	case CmdPause:
		sig = syscall.SIGUSR1
	case CmdResume:
		sig = syscall.SIGUSR2
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
	return syscall.Kill(pid, sig)
}

func (c *signalClient) Ping() error {
	pid, err := ReadPID(c.snapDir)
	if err != nil {
		return err
	}
	return syscall.Kill(pid, 0)
}
