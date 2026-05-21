package input

import (
	"encoding/json"
	"time"
)

type State struct {
	DeviceID  string          `json:"device_id"`
	Timestamp string          `json:"timestamp,omitempty"`
	Buttons   map[string]bool `json:"buttons"`
	Type      string          `json:"type,omitempty"`
}

type Provider interface {
	ID() string
	Send(s *State) error
}

type MockProvider struct {
	id   string
	emit func([]byte) error
}

func NewMockProvider(id string, emit func([]byte) error) *MockProvider {
	return &MockProvider{id: id, emit: emit}
}

func (m *MockProvider) ID() string { return m.id }

func (m *MockProvider) Send(s *State) error {
	if s.Timestamp == "" {
		s.Timestamp = time.Now().Format(time.RFC3339)
	}
	if s.DeviceID == "" {
		s.DeviceID = m.id
	}
	s.Type = "input"
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if m.emit != nil {
		return m.emit(b)
	}
	return nil
}
