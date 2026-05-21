package input

import (
	"encoding/json"
	"time"
)

type GamepadProvider struct {
	id   string
	emit func([]byte) error
}

func NewGamepadProvider(id string, emit func([]byte) error) *GamepadProvider {
	return &GamepadProvider{id: id, emit: emit}
}

func (g *GamepadProvider) ID() string { return g.id }

func (g *GamepadProvider) Send(s *State) error {
	if s.Timestamp == "" {
		s.Timestamp = time.Now().Format(time.RFC3339)
	}
	if s.DeviceID == "" {
		s.DeviceID = g.id
	}
	s.Type = "input"
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if g.emit != nil {
		return g.emit(b)
	}
	return nil
}
