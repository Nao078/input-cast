//go:build windows
// +build windows

package gamepad

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"leverless-overlay/internal/bridge"
)

const (
	xinputMaxControllers = 4
	xinputPollInterval   = 16 * time.Millisecond
	xinputTriggerPressed = 30
	xinputStickPressed   = 10000

	xinputGamepadDpadUp        = 0x0001
	xinputGamepadDpadDown      = 0x0002
	xinputGamepadDpadLeft      = 0x0004
	xinputGamepadDpadRight     = 0x0008
	xinputGamepadStart         = 0x0010
	xinputGamepadBack          = 0x0020
	xinputGamepadLeftThumb     = 0x0040
	xinputGamepadRightThumb    = 0x0080
	xinputGamepadLeftShoulder  = 0x0100
	xinputGamepadRightShoulder = 0x0200
	xinputGamepadA             = 0x1000
	xinputGamepadB             = 0x2000
	xinputGamepadX             = 0x4000
	xinputGamepadY             = 0x8000

	winErrorSuccess            = 0
	winErrorDeviceNotConnected = 1167
)

type windowsXInputBackend struct {
	getState *syscall.LazyProc
}

type xinputState struct {
	PacketNumber uint32
	Gamepad      xinputGamepad
}

type xinputGamepad struct {
	Buttons      uint16
	LeftTrigger  byte
	RightTrigger byte
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

func newPlatformBackend(scanInterval time.Duration) Backend {
	return &windowsXInputBackend{
		getState: loadXInputGetState(),
	}
}

func (b *windowsXInputBackend) Name() string {
	return "Windows XInput"
}

func (b *windowsXInputBackend) Run(ctx context.Context, updates chan<- Snapshot) error {
	if b.getState == nil {
		return fmt.Errorf("XInput is not available")
	}

	ticker := time.NewTicker(xinputPollInterval)
	defer ticker.Stop()

	connected := [xinputMaxControllers]bool{}
	lastButtons := [xinputMaxControllers]map[string]bool{}

	for {
		changed := b.poll(updates, &connected, &lastButtons)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if !changed {
				continue
			}
		}
	}
}

func (b *windowsXInputBackend) poll(updates chan<- Snapshot, connected *[xinputMaxControllers]bool, lastButtons *[xinputMaxControllers]map[string]bool) bool {
	anyChanged := false
	for index := 0; index < xinputMaxControllers; index++ {
		state, code := b.getControllerState(index)
		deviceName := fmt.Sprintf("XInput Controller #%d", index+1)
		if code == winErrorDeviceNotConnected {
			if connected[index] {
				connected[index] = false
				lastButtons[index] = nil
				updates <- Snapshot{DeviceName: deviceName, Message: deviceName + " disconnected"}
				anyChanged = true
			}
			continue
		}
		if code != winErrorSuccess {
			updates <- Snapshot{DeviceName: deviceName, Err: fmt.Errorf("XInputGetState failed: %d", code)}
			continue
		}
		if !connected[index] {
			connected[index] = true
			updates <- Snapshot{DeviceName: deviceName, Message: deviceName + " connected"}
		}

		buttons := xinputButtons(state.Gamepad)
		if !sameButtons(lastButtons[index], buttons) {
			lastButtons[index] = bridge.CloneButtons(buttons)
			updates <- newSnapshot(deviceName, buttons)
			anyChanged = true
		}
	}
	return anyChanged
}

func (b *windowsXInputBackend) getControllerState(index int) (xinputState, uintptr) {
	var state xinputState
	ret, _, _ := b.getState.Call(uintptr(index), uintptr(unsafe.Pointer(&state)))
	return state, ret
}

func xinputButtons(gamepad xinputGamepad) map[string]bool {
	buttons := bridge.NewButtons()
	flags := gamepad.Buttons
	buttons["up"] = flags&xinputGamepadDpadUp != 0 || gamepad.ThumbLY > xinputStickPressed
	buttons["down"] = flags&xinputGamepadDpadDown != 0 || gamepad.ThumbLY < -xinputStickPressed
	buttons["left"] = flags&xinputGamepadDpadLeft != 0 || gamepad.ThumbLX < -xinputStickPressed
	buttons["right"] = flags&xinputGamepadDpadRight != 0 || gamepad.ThumbLX > xinputStickPressed
	buttons["b1"] = flags&xinputGamepadA != 0
	buttons["b2"] = flags&xinputGamepadB != 0
	buttons["b3"] = flags&xinputGamepadX != 0
	buttons["b4"] = flags&xinputGamepadY != 0
	buttons["l1"] = flags&xinputGamepadLeftShoulder != 0
	buttons["r1"] = flags&xinputGamepadRightShoulder != 0
	buttons["l2"] = gamepad.LeftTrigger > xinputTriggerPressed
	buttons["r2"] = gamepad.RightTrigger > xinputTriggerPressed
	buttons["s1"] = flags&xinputGamepadBack != 0
	buttons["s2"] = flags&xinputGamepadStart != 0
	buttons["l3"] = flags&xinputGamepadLeftThumb != 0
	buttons["r3"] = flags&xinputGamepadRightThumb != 0
	return buttons
}

func sameButtons(a, b map[string]bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func loadXInputGetState() *syscall.LazyProc {
	for _, dllName := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
		proc := syscall.NewLazyDLL(dllName).NewProc("XInputGetState")
		if proc.Find() == nil {
			return proc
		}
	}
	return nil
}
