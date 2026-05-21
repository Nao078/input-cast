//go:build !linux

package input

type EvdevProvider struct {
	id string
}

func NewEvdevProvider(id string, emit func([]byte) error) *EvdevProvider {
	return &EvdevProvider{id: id}
}

func (e *EvdevProvider) ID() string { return e.id }

func (e *EvdevProvider) Send(s *State) error { return nil }
