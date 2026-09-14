// Package ipc implements soute's transport-agnostic request/response protocol
// over net.Conn. Both Unix domain sockets (darwin/linux) and go-winio named
// pipes (windows) expose net.Listener/net.Conn, so the client, server, and
// wire format are shared; only endpoint resolution and the ACL are
// platform-specific.
package ipc

import (
	"encoding/json"
)

// ProtocolVersion gates the wire protocol. The daemon rejects envelopes with a
// different major version.
const ProtocolVersion = 1

// Command names.
const (
	CmdPing   = "ping"
	CmdStatus = "status"
	CmdPause  = "pause"
	CmdResume = "resume"
	CmdStop   = "stop"
	CmdPrune  = "prune"
)

// Envelope is a client-to-daemon request.
type Envelope struct {
	Version int             `json:"version"`
	ID      uint64          `json:"id"` // echoed back for correlation
	Cmd     string          `json:"cmd"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response is a daemon-to-client reply.
type Response struct {
	Version int             `json:"version"`
	ID      uint64          `json:"id"`
	OK      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// PingResponse is the payload of a successful ping.
type PingResponse struct {
	UptimeSeconds float64 `json:"uptime_seconds"`
	Paused        bool    `json:"paused"`
	PID           int     `json:"pid"`
}

// StatusResponse is the payload of a successful status command.
type StatusResponse struct {
	Status           string  `json:"status"`
	Paused           bool    `json:"paused"`
	UptimeSeconds    float64 `json:"uptime_seconds"`
	PID              int     `json:"pid"`
	TargetPath       string  `json:"target_path"`
	LastSnapshotID   string  `json:"last_snapshot_id"`
	LastSnapshotTime string  `json:"last_snapshot_time"`
}

// PauseResponse is the payload of a successful pause command.
type PauseResponse struct {
	WasPaused bool `json:"was_paused"`
}

// ResumeResponse is the payload of a successful resume command.
type ResumeResponse struct {
	BaselineMtime string `json:"baseline_mtime"`
	BaselineHash  string `json:"baseline_hash"`
}

// PruneResponse is the payload of a successful prune command.
type PruneResponse struct {
	Removed      int `json:"removed"`
	DeletedBlobs int `json:"deleted_blobs"`
}

// newResponse builds a response envelope from a payload or error.
func newResponse(id uint64, ok bool, payload any, err error) Response {
	r := Response{Version: ProtocolVersion, ID: id, OK: ok}
	if err != nil {
		r.Error = err.Error()
		return r
	}
	if payload != nil {
		b, _ := json.Marshal(payload)
		r.Payload = b
	}
	return r
}
