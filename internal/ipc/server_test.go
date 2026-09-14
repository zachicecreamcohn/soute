package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"
)

func callCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestCallRoundTrip(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	h := HandlerFunc(func(cmd string, _ json.RawMessage) (any, error) {
		return map[string]any{"echo": cmd}, nil
	})
	go handleConn(serverConn, h)

	c := NewClient(clientConn)
	raw, err := c.Call(callCtx(t), "ping", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["echo"] != "ping" {
		t.Errorf("echo = %v want ping", m["echo"])
	}
}

func TestCallError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	h := HandlerFunc(func(string, json.RawMessage) (any, error) {
		return nil, errors.New("boom")
	})
	go handleConn(serverConn, h)

	c := NewClient(clientConn)
	_, err := c.Call(callCtx(t), "bad", nil)
	if err == nil || err.Error() != "boom" {
		t.Errorf("err = %v want boom", err)
	}
}

func TestCallCorrelatesIDs(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	h := HandlerFunc(func(cmd string, _ json.RawMessage) (any, error) {
		return cmd, nil
	})
	go handleConn(serverConn, h)

	c := NewClient(clientConn)
	// Two sequential calls correlate to the right responses.
	for _, cmd := range []string{"first", "second"} {
		raw, err := c.Call(callCtx(t), cmd, nil)
		if err != nil {
			t.Fatalf("Call(%q): %v", cmd, err)
		}
		var got string
		if err := json.Unmarshal(raw, &got); err != nil || got != cmd {
			t.Errorf("got %q want %q", got, cmd)
		}
	}
}

func TestVersionGate(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	h := HandlerFunc(func(string, json.RawMessage) (any, error) {
		t.Error("handler should not run for wrong version")
		return nil, nil
	})
	go handleConn(serverConn, h)

	// Send an envelope with a bogus version directly.
	enc := json.NewEncoder(clientConn)
	dec := json.NewDecoder(clientConn)
	if err := enc.Encode(Envelope{Version: 99, ID: 1, Cmd: CmdPing}); err != nil {
		t.Fatal(err)
	}
	var resp Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Error("expected version rejection")
	}
}
