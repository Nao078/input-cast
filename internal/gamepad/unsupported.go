//go:build !linux && !windows
// +build !linux,!windows

package gamepad

import (
	"context"
	"fmt"
	"time"
)

type unsupportedBackend struct{}

func newPlatformBackend(scanInterval time.Duration) Backend {
	return unsupportedBackend{}
}

func (unsupportedBackend) Name() string {
	return "Unsupported"
}

func (unsupportedBackend) Run(ctx context.Context, updates chan<- Snapshot) error {
	return fmt.Errorf("input-cast-bridge currently supports Linux and Windows clients only")
}
