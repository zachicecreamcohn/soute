package ipc

import (
	"encoding/json"
	"fmt"
	"net"
)

// Handler processes a command and returns a payload or error. The daemon's
// watcher handler implements it.
type Handler interface {
	Handle(cmd string, payload json.RawMessage) (any, error)
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(cmd string, payload json.RawMessage) (any, error)

// Handle implements Handler.
func (f HandlerFunc) Handle(cmd string, payload json.RawMessage) (any, error) {
	return f(cmd, payload)
}

// Serve accepts connections on l until the listener closes, serving each on
// its own goroutine.
func Serve(l net.Listener, h Handler) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go handleConn(conn, h)
	}
}

// handleConn serves a single connection until it closes or a stop command is
// processed. Framing is newline-delimited JSON (one envelope per line).
func handleConn(conn net.Conn, h Handler) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req Envelope
		if err := dec.Decode(&req); err != nil {
			return
		}
		if req.Version != ProtocolVersion {
			_ = enc.Encode(newResponse(req.ID, false, nil, fmt.Errorf("unsupported protocol version %d", req.Version)))
			continue
		}
		payload, err := h.Handle(req.Cmd, req.Payload)
		if err := enc.Encode(newResponse(req.ID, err == nil, payload, err)); err != nil {
			return
		}
		if req.Cmd == CmdStop {
			return
		}
	}
}
