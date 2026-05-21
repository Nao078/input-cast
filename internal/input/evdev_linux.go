//go:build linux
// +build linux

package input

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	evdev "github.com/gvalkov/golang-evdev"
)

type EvdevProvider struct {
	id      string
	emit    func([]byte) error
	mu      sync.Mutex
	watched map[string]bool
}

func NewEvdevProvider(id string, emit func([]byte) error) *EvdevProvider {
	p := &EvdevProvider{id: id, emit: emit, watched: make(map[string]bool)}
	go p.run()
	return p
}

func (e *EvdevProvider) ID() string { return e.id }

func (e *EvdevProvider) Send(s *State) error {
	// Not used for automatic device reading; accept manual sends
	if s.Timestamp == "" {
		s.Timestamp = time.Now().Format(time.RFC3339)
	}
	if s.DeviceID == "" {
		s.DeviceID = e.id
	}
	s.Type = "input"
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if e.emit != nil {
		return e.emit(b)
	}
	return nil
}

func (e *EvdevProvider) run() {
	for {
		paths, _ := filepath.Glob("/dev/input/event*")
		for _, p := range paths {
			e.mu.Lock()
			if e.watched[p] {
				e.mu.Unlock()
				continue
			}
			e.watched[p] = true
			e.mu.Unlock()
			go e.watchDevice(p)
		}
		time.Sleep(2 * time.Second)
	}
}

func (e *EvdevProvider) watchDevice(path string) {
	dev, err := evdev.Open(path)
	if err != nil {
		e.mu.Lock()
		delete(e.watched, path)
		e.mu.Unlock()
		time.Sleep(2 * time.Second)
		return
	}
	name := strings.TrimSpace(dev.Name)
	// prepare device-specific mapping if available
	mapping := deviceMapping(name)
	state := make(map[string]bool)
	for {
		evs, err := dev.Read()
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		// only handle key and abs for now
		buttons := make(map[string]bool)
		for _, ev := range evs {
			if ev.Type == evdev.EV_KEY {
				// map common gamepad/keyboard keys
				pressed := ev.Value > 0
				if lbl, ok := mapping[int(ev.Code)]; ok {
					buttons[lbl] = pressed
					state[lbl] = pressed
					continue
				}
				switch ev.Code {
				case evdev.BTN_SOUTH: // A
					buttons["b1"] = pressed
				case evdev.BTN_EAST: // B
					buttons["b2"] = pressed
				case evdev.BTN_NORTH: // Y
					buttons["b3"] = pressed
				case evdev.BTN_WEST: // X
					buttons["b4"] = pressed
				case evdev.BTN_TL:
					buttons["l1"] = pressed
				case evdev.BTN_TR:
					buttons["r1"] = pressed
				case evdev.BTN_TL2:
					buttons["l2"] = pressed
				case evdev.BTN_TR2:
					buttons["r2"] = pressed
				case evdev.BTN_SELECT:
					buttons["s1"] = pressed
				case evdev.BTN_START:
					buttons["s2"] = pressed
				case evdev.BTN_THUMBL:
					buttons["l3"] = pressed
				case evdev.BTN_THUMBR:
					buttons["r3"] = pressed
				default:
					// fallback to code_N
					buttons[fmt.Sprintf("code_%d", ev.Code)] = pressed
				}
				for id, pressed := range buttons {
					state[id] = pressed
				}
			} else if ev.Type == evdev.EV_ABS {
				// map common hat/axes to directions
				switch ev.Code {
				case evdev.ABS_HAT0X:
					buttons["left"] = ev.Value < 0
					buttons["right"] = ev.Value > 0
					state["left"] = buttons["left"]
					state["right"] = buttons["right"]
				case evdev.ABS_HAT0Y:
					buttons["up"] = ev.Value < 0
					buttons["down"] = ev.Value > 0
					state["up"] = buttons["up"]
					state["down"] = buttons["down"]
				case evdev.ABS_X:
					// analog stick: threshold
					buttons["left"] = ev.Value < -10000
					buttons["right"] = ev.Value > 10000
					state["left"] = buttons["left"]
					state["right"] = buttons["right"]
				case evdev.ABS_Y:
					buttons["up"] = ev.Value < -10000
					buttons["down"] = ev.Value > 10000
					state["up"] = buttons["up"]
					state["down"] = buttons["down"]
				default:
					// generic abs
					buttons[fmt.Sprintf("abs_%d", ev.Code)] = ev.Value != 0
				}
			}
		}

		if len(buttons) > 0 {
			st := &State{DeviceID: e.id, Buttons: cloneBoolMap(state)}
			_ = e.Send(st)
		}
	}
}

func cloneBoolMap(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// deviceMapping returns a mapping from evdev codes to internal button ids for known devices.
func deviceMapping(name string) map[int]string {
	lname := strings.ToLower(name)
	m := make(map[int]string)
	// GP2040-CE / OpenStick conventions - example mapping
	if strings.Contains(lname, "gp2040") || strings.Contains(lname, "openstick") || strings.Contains(lname, "gp2040-ce") {
		m[int(evdev.BTN_SOUTH)] = "b1"
		m[int(evdev.BTN_EAST)] = "b2"
		m[int(evdev.BTN_NORTH)] = "b3"
		m[int(evdev.BTN_WEST)] = "b4"
		m[int(evdev.BTN_TL)] = "l1"
		m[int(evdev.BTN_TR)] = "r1"
		m[int(evdev.BTN_TL2)] = "l2"
		m[int(evdev.BTN_TR2)] = "r2"
		m[int(evdev.BTN_SELECT)] = "s1"
		m[int(evdev.BTN_START)] = "s2"
		m[int(evdev.BTN_THUMBL)] = "l3"
		m[int(evdev.BTN_THUMBR)] = "r3"
	}
	return m
}
