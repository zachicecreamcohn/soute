package ipc

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	e := Envelope{Version: ProtocolVersion, ID: 42, Cmd: CmdPing, Payload: json.RawMessage(`{}`)}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got Envelope
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.Cmd != CmdPing {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestResponseOK(t *testing.T) {
	r := newResponse(7, true, map[string]any{"a": 1}, nil)
	if !r.OK || r.Error != "" {
		t.Errorf("expected OK response: %+v", r)
	}
	var m map[string]int
	if err := json.Unmarshal(r.Payload, &m); err != nil || m["a"] != 1 {
		t.Errorf("payload = %s", r.Payload)
	}
}

func TestResponseError(t *testing.T) {
	r := newResponse(7, false, nil, errTest{msg: "boom"})
	if r.OK || r.Error != "boom" || len(r.Payload) != 0 {
		t.Errorf("expected error response: %+v", r)
	}
}

type errTest struct{ msg string }

func (e errTest) Error() string { return e.msg }
