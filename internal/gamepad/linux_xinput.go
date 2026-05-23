//go:build linux
// +build linux

package gamepad

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"leverless-overlay/internal/bridge"

	evdev "github.com/gvalkov/golang-evdev"
)

type linuxXInputBackend struct {
	mu           sync.Mutex
	watched      map[string]bool
	scanInterval time.Duration
}

func newPlatformBackend(scanInterval time.Duration) Backend {
	return &linuxXInputBackend{
		watched:      make(map[string]bool),
		scanInterval: scanInterval,
	}
}

func (b *linuxXInputBackend) Name() string {
	return "Linux Input"
}

func (b *linuxXInputBackend) Run(ctx context.Context, updates chan<- Snapshot) error {
	ticker := time.NewTicker(b.scanInterval)
	defer ticker.Stop()

	if err := b.scan(ctx, updates); err != nil {
		updates <- Snapshot{Err: err}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := b.scan(ctx, updates); err != nil {
				updates <- Snapshot{Err: err}
			}
		}
	}
}

func (b *linuxXInputBackend) scan(ctx context.Context, updates chan<- Snapshot) error {
	paths, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no /dev/input/event* devices found")
	}

	var lastErr error
	for _, path := range paths {
		b.mu.Lock()
		if b.watched[path] {
			b.mu.Unlock()
			continue
		}
		b.watched[path] = true
		b.mu.Unlock()

		dev, err := evdev.Open(path)
		if err != nil {
			b.unwatch(path)
			if os.IsPermission(err) {
				lastErr = fmt.Errorf("permission denied opening %s; add your user to the input group and log in again", path)
				continue
			}
			lastErr = err
			continue
		}
		if !looksLikeXInputGamepad(dev) {
			_ = dev.File.Close()
			continue
		}
		go b.watch(ctx, path, dev, updates)
	}
	return lastErr
}

func (b *linuxXInputBackend) watch(ctx context.Context, path string, dev *evdev.InputDevice, updates chan<- Snapshot) {
	defer b.unwatch(path)
	defer dev.File.Close()

	deviceName := strings.TrimSpace(dev.Name)
	if deviceName == "" {
		deviceName = path
	}

	buttons := bridge.NewButtons()
	unknown := make(map[string]bool)
	updates <- newSnapshot(deviceName, buttons)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		events, err := dev.Read()
		if err != nil {
			updates <- Snapshot{DeviceName: deviceName, Err: err}
			time.Sleep(500 * time.Millisecond)
			return
		}

		changed := false
		for _, event := range events {
			switch event.Type {
			case evdev.EV_KEY:
				if id, ok := keyCodeToButton(event.Code); ok {
					buttons[id] = event.Value > 0
					changed = true
				} else if event.Value > 0 {
					emitUnknownEvent(deviceName, unknown, updates, "EV_KEY", event.Code, event.Value)
				}
			case evdev.EV_ABS:
				if applyAbsEvent(buttons, event.Code, event.Value) {
					changed = true
				} else if event.Value != 0 {
					emitUnknownEvent(deviceName, unknown, updates, "EV_ABS", event.Code, event.Value)
				}
			}
		}

		if changed {
			updates <- newSnapshot(deviceName, buttons)
		}
	}
}

func emitUnknownEvent(deviceName string, unknown map[string]bool, updates chan<- Snapshot, eventType string, code uint16, value int32) {
	key := fmt.Sprintf("%s:%d", eventType, code)
	if unknown[key] {
		return
	}
	unknown[key] = true
	updates <- Snapshot{
		DeviceName: deviceName,
		Message:    fmt.Sprintf("unmapped %s code=%d value=%d", eventType, code, value),
	}
}

func (b *linuxXInputBackend) unwatch(path string) {
	b.mu.Lock()
	delete(b.watched, path)
	b.mu.Unlock()
}

func looksLikeXInputGamepad(dev *evdev.InputDevice) bool {
	name := strings.ToLower(dev.Name)
	if strings.Contains(name, "x-box") ||
		strings.Contains(name, "xbox") ||
		strings.Contains(name, "xinput") ||
		strings.Contains(name, "gp2040") ||
		strings.Contains(name, "openstick") ||
		strings.Contains(name, "playstation") ||
		strings.Contains(name, "dualsense") ||
		strings.Contains(name, "dualshock") ||
		strings.Contains(name, "switch") ||
		strings.Contains(name, "nintendo") {
		return true
	}

	hasGamepadButton := false
	hasAbs := false
	for capType, codes := range dev.Capabilities {
		switch capType.Type {
		case evdev.EV_KEY:
			for _, code := range codes {
				switch code.Code {
				case evdev.BTN_SOUTH, evdev.BTN_EAST, evdev.BTN_NORTH, evdev.BTN_WEST:
					hasGamepadButton = true
				}
			}
		case evdev.EV_ABS:
			for _, code := range codes {
				switch code.Code {
				case evdev.ABS_X, evdev.ABS_Y, evdev.ABS_HAT0X, evdev.ABS_HAT0Y:
					hasAbs = true
				}
			}
		}
	}
	return hasGamepadButton && hasAbs
}

func keyCodeToButton(code uint16) (string, bool) {
	switch code {
	case evdev.BTN_SOUTH:
		return "b1", true
	case evdev.BTN_EAST:
		return "b2", true
	case evdev.BTN_NORTH:
		return "b3", true
	case evdev.BTN_WEST:
		return "b4", true
	case evdev.BTN_TRIGGER:
		return "b1", true
	case evdev.BTN_THUMB:
		return "b2", true
	case evdev.BTN_THUMB2:
		return "b3", true
	case evdev.BTN_TOP:
		return "b4", true
	case evdev.BTN_TL:
		return "l1", true
	case evdev.BTN_TR:
		return "r1", true
	case evdev.BTN_TL2:
		return "l2", true
	case evdev.BTN_TR2:
		return "r2", true
	case evdev.BTN_TOP2:
		return "l1", true
	case evdev.BTN_PINKIE:
		return "r1", true
	case evdev.BTN_BASE:
		return "l2", true
	case evdev.BTN_BASE2:
		return "r2", true
	case evdev.BTN_SELECT:
		return "s1", true
	case evdev.BTN_START:
		return "s2", true
	case evdev.BTN_BASE3:
		return "s1", true
	case evdev.BTN_BASE4:
		return "s2", true
	case evdev.BTN_THUMBL:
		return "l3", true
	case evdev.BTN_THUMBR:
		return "r3", true
	case evdev.BTN_MODE:
		return "a1", true
	case evdev.KEY_UP:
		return "up", true
	case evdev.KEY_DOWN:
		return "down", true
	case evdev.KEY_LEFT:
		return "left", true
	case evdev.KEY_RIGHT:
		return "right", true
	case evdev.KEY_W:
		return "up", true
	case evdev.KEY_S:
		return "down", true
	case evdev.KEY_A:
		return "left", true
	case evdev.KEY_D:
		return "right", true
	case evdev.KEY_Z:
		return "b1", true
	case evdev.KEY_X:
		return "b2", true
	case evdev.KEY_C:
		return "b3", true
	case evdev.KEY_V:
		return "b4", true
	case evdev.KEY_Q:
		return "l1", true
	case evdev.KEY_E:
		return "r1", true
	case evdev.KEY_R:
		return "l2", true
	case evdev.KEY_F:
		return "r2", true
	case evdev.KEY_ENTER:
		return "s2", true
	case evdev.KEY_BACKSPACE:
		return "s1", true
	case evdev.KEY_ESC:
		return "a1", true
	case evdev.KEY_SPACE:
		return "a2", true
	}
	return "", false
}

func applyAbsEvent(buttons map[string]bool, code uint16, value int32) bool {
	beforeLeft := buttons["left"]
	beforeRight := buttons["right"]
	beforeUp := buttons["up"]
	beforeDown := buttons["down"]
	beforeL2 := buttons["l2"]
	beforeR2 := buttons["r2"]

	switch code {
	case evdev.ABS_HAT0X:
		buttons["left"] = value < 0
		buttons["right"] = value > 0
	case evdev.ABS_HAT0Y:
		buttons["up"] = value < 0
		buttons["down"] = value > 0
	case evdev.ABS_X:
		buttons["left"] = value < -10000
		buttons["right"] = value > 10000
	case evdev.ABS_Y:
		buttons["up"] = value < -10000
		buttons["down"] = value > 10000
	case evdev.ABS_Z, evdev.ABS_BRAKE:
		buttons["l2"] = value > 0
	case evdev.ABS_RZ, evdev.ABS_GAS:
		buttons["r2"] = value > 0
	default:
		return false
	}

	return beforeLeft != buttons["left"] ||
		beforeRight != buttons["right"] ||
		beforeUp != buttons["up"] ||
		beforeDown != buttons["down"] ||
		beforeL2 != buttons["l2"] ||
		beforeR2 != buttons["r2"]
}
