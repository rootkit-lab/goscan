package scanhub

import (
	"encoding/json"
	"fmt"
)

const (
	TypeHello     = "hello"
	TypeProgress  = "progress"
	TypeFound     = "found"
	TypeFoundBody = "found_body"
	TypeDone      = "done"
	TypePing      = "ping"
	TypePong      = "pong"
	TypeError     = "error"
)

// Message is a hub protocol frame (JSON over WebSocket).
type Message struct {
	Type           string `json:"t"`
	RunID          string `json:"runId,omitempty"`
	WorkerID       string `json:"workerId,omitempty"`
	Token          string `json:"token,omitempty"`
	Total          int64  `json:"total,omitempty"`
	Scanned        int64  `json:"scanned,omitempty"`
	Vulns          int64  `json:"vulns,omitempty"`
	Rate           int64  `json:"rate,omitempty"`
	Domain         string `json:"domain,omitempty"`
	Path           string `json:"path,omitempty"`
	URL            string `json:"url,omitempty"`
	Confidence     string `json:"confidence,omitempty"`
	HasCredentials bool   `json:"hasCredentials,omitempty"`
	ContentB64     string `json:"contentB64,omitempty"`
	Compressed     bool   `json:"compressed,omitempty"`
	OK             bool   `json:"ok,omitempty"`
	Error          string `json:"error,omitempty"`
}

func Encode(m Message) ([]byte, error) {
	return json.Marshal(m)
}

func Decode(data []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return Message{}, err
	}
	if m.Type == "" {
		return Message{}, fmt.Errorf("tipo em falta")
	}
	return m, nil
}
