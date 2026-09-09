//go:build windows

package ipc

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
)

const (
	pingTimeout  = 500 * time.Millisecond
	sendTimeout  = 3 * time.Second
)

type pipeServer struct {
	l    net.Listener
	cmds chan Command
}

// NewServer listens on a per-project named pipe for control commands.
func NewServer(snapDir string) (Server, error) {
	l, err := winio.ListenPipe(`\\.\pipe\`+PipeName(snapDir), &winio.PipeConfig{})
	if err != nil {
		return nil, err
	}
	s := &pipeServer{l: l, cmds: make(chan Command, 8)}
	go s.acceptLoop()
	return s, nil
}

func (s *pipeServer) acceptLoop() {
	for {
		conn, err := s.l.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *pipeServer) handle(conn net.Conn) {
	defer conn.Close()
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return
	}
	cmd := Command(strings.TrimSpace(line))
	switch cmd {
	case CmdStop, CmdPause, CmdResume:
		s.cmds <- cmd
		fmt.Fprintln(conn, "ok")
	}
}

func (s *pipeServer) Commands() <-chan Command { return s.cmds }

func (s *pipeServer) Close() error { return s.l.Close() }

type pipeClient struct {
	pipeName string
}

func NewClient(snapDir string) Client {
	return &pipeClient{pipeName: `\\.\pipe\` + PipeName(snapDir)}
}

func (c *pipeClient) Send(cmd Command) error {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, c.pipeName)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := fmt.Fprintf(conn, "%s\n", cmd); err != nil {
		return err
	}
	_, err = bufio.NewReader(conn).ReadString('\n')
	return err
}

func (c *pipeClient) Ping() error {
	timeout := pingTimeout
	conn, err := winio.DialPipe(c.pipeName, &timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
