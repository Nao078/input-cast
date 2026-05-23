package gamepad

import (
	"context"
	"time"

	"leverless-overlay/internal/bridge"
)

type Snapshot struct {
	DeviceName string
	Buttons    map[string]bool
	Message    string
	Err        error
}

type Backend interface {
	Name() string
	Run(ctx context.Context, updates chan<- Snapshot) error
}

func NewBackend(scanInterval time.Duration) Backend {
	if scanInterval <= 0 {
		scanInterval = 2 * time.Second
	}
	return newPlatformBackend(scanInterval)
}

func newSnapshot(deviceName string, buttons map[string]bool) Snapshot {
	return Snapshot{
		DeviceName: deviceName,
		Buttons:    bridge.CloneButtons(buttons),
	}
}
