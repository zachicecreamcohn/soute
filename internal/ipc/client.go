package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client issues requests over a single duplex net.Conn.
type Client struct {
	conn   net.Conn
	enc    *json.Encoder
	dec    *json.Decoder
	mu     sync.Mutex // serialize requests (one in flight at a time)
	nextID uint64
}

// NewClient wraps an established connection.
func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn, enc: json.NewEncoder(conn), dec: json.NewDecoder(conn)}
}

// Call sends cmd and waits for its correlated response. It returns the raw
// payload on success, or an error on transport failure or a daemon-reported
// error.
func (c *Client) Call(ctx context.Context, cmd string, payload any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	id := c.nextID

	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		raw = b
	}

	if err := c.enc.Encode(Envelope{Version: ProtocolVersion, ID: id, Cmd: cmd, Payload: raw}); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	deadline := time.Now().Add(readTimeout)
	if dl, ok := ctx.Deadline(); ok {
		deadline = dl
	}
	_ = c.conn.SetReadDeadline(deadline)
	defer c.conn.SetReadDeadline(time.Time{})

	var resp Response
	if err := c.dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("receive: %w", err)
	}
	if resp.ID != id {
		return nil, fmt.Errorf("response id mismatch: got %d want %d", resp.ID, id)
	}
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Payload, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error { return c.conn.Close() }

// DialClient connects to addr using the platform transport and wraps the
// connection in a Client.
func DialClient(ctx context.Context, addr string) (*Client, error) {
	conn, err := Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), nil
}
