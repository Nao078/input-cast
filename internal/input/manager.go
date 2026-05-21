package input

import (
	"encoding/json"
	"sync"
	"time"
)

type providerEntry struct {
	prov    Provider
	enabled bool
}

type Manager struct {
	mu        sync.Mutex
	providers map[string]providerEntry
	subs      []func(*State)
	last      map[string]*State
	events    map[string][]*State
}

func NewManager() *Manager {
	return &Manager{providers: make(map[string]providerEntry), last: make(map[string]*State), events: make(map[string][]*State)}
}

func (m *Manager) Register(p Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.ID()] = providerEntry{prov: p, enabled: true}
}

func (m *Manager) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.providers, id)
	delete(m.last, id)
}

func (m *Manager) Providers() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.providers))
	for id := range m.providers {
		out = append(out, id)
	}
	return out
}

func (m *Manager) SetEnabled(id string, en bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.providers[id]; ok {
		e.enabled = en
		m.providers[id] = e
	}
}

func (m *Manager) Enabled(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.providers[id]; ok {
		return e.enabled
	}
	return false
}

type ProviderInfo struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	Last    *State `json:"last_state,omitempty"`
}

func (m *Manager) ProvidersInfo() []ProviderInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ProviderInfo, 0, len(m.providers))
	for id, e := range m.providers {
		pi := ProviderInfo{ID: id, Enabled: e.enabled}
		if ls, ok := m.last[id]; ok {
			pi.Last = ls
		}
		out = append(out, pi)
	}
	return out
}

func (m *Manager) Subscribe(fn func(*State)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subs = append(m.subs, fn)
}

// HandleRaw accepts raw JSON bytes from providers, decodes to State and notifies subscribers.
func (m *Manager) HandleRaw(b []byte) error {
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.Timestamp == "" {
		s.Timestamp = time.Now().Format(time.RFC3339)
	}
	if s.DeviceID != "" && !m.enabledForDevice(s.DeviceID) {
		return nil
	}
	// store last state by device id if available
	if s.DeviceID != "" {
		m.mu.Lock()
		// store copy
		copy := s
		m.last[s.DeviceID] = &copy
		// append to events (keep last 200)
		arr := m.events[s.DeviceID]
		arr = append(arr, &copy)
		if len(arr) > 200 {
			arr = arr[len(arr)-200:]
		}
		m.events[s.DeviceID] = arr
		m.mu.Unlock()
	}
	m.notify(&s)
	return nil
}

func (m *Manager) enabledForDevice(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.providers[id]
	if !ok {
		return true
	}
	return entry.enabled
}

func (m *Manager) Events(id string) []*State {
	m.mu.Lock()
	defer m.mu.Unlock()
	arr := m.events[id]
	out := make([]*State, len(arr))
	copy(out, arr)
	return out
}

func (m *Manager) notify(s *State) {
	m.mu.Lock()
	subs := append([]func(*State){}, m.subs...)
	m.mu.Unlock()
	for _, fn := range subs {
		go fn(s)
	}
}
