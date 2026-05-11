package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/it-easy/stagelinq-go/internal/debug"
	"github.com/it-easy/stagelinq-go/network/discovery"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	logger := debug.New(debug.Info)

	listener := discovery.NewListener(logger)

	devices, err := listener.Listen(ctx)
	if err != nil {
		logger.Error("failed to start discovery", "error", err)
		os.Exit(1)
	}

	seen := make(map[string]bool)

	for device := range devices {
		if device.SoftwareName == "OfflineAnalyzer" {
			continue
		}

		key := device.TokenHex + "@" + device.IP.String()

		if seen[key] {
			continue
		}

		seen[key] = true

		logger.Info(
			"device discovered",
			"source", device.Source,
			"action", device.Action,
			"software", device.SoftwareName,
			"version", device.SoftwareVersion,
			"ip", device.IP.String(),
			"port", device.Port,
			"token", device.TokenHex,
		)
	}
}
