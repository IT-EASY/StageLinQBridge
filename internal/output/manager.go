// Package output manages all protocol senders and dispatches beat events.
package output

import (
	"sync/atomic"

	"github.com/it-easy/StageLinQBridge/internal/config"
	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/output/artnet"
	"github.com/it-easy/StageLinQBridge/internal/output/osc"
	"github.com/it-easy/StageLinQBridge/internal/output/sacn"
)

// sender is a minimal interface implemented by all protocol senders.
type sender interface {
	Beat()
	Downbeat()
	Slow()
	Close()
}

// Manager holds all configured senders and routes events to enabled ones.
type Manager struct {
	logger     *debug.Logger
	sacnSender *sacn.Sender
	artSender  *artnet.Sender
	oscSender  *osc.Sender

	// Runtime enable flags (toggled from UI without restart).
	sacnOn int32 // atomic bool
	artOn  int32
	oscOn  int32
}

// New initialises all senders that are configured and enabled.
// Senders that fail to initialise are logged and skipped (non-fatal).
func New(cfg *config.Config, logger *debug.Logger) *Manager {
	m := &Manager{logger: logger}

	if cfg.SACN.Universe > 0 {
		s, err := sacn.New(
			cfg.SACN.Universe,
			cfg.SACN.Channels.Beat,
			cfg.SACN.Channels.Downbeat,
			cfg.SACN.Channels.Slow,
			cfg.SACN.PulseMS,
		)
		if err != nil {
			logger.Warn("sACN sender init failed", "error", err)
		} else {
			m.sacnSender = s
			if cfg.SACN.Enabled {
				atomic.StoreInt32(&m.sacnOn, 1)
			}
			logger.Info("sACN sender ready",
				"universe", cfg.SACN.Universe,
				"enabled", cfg.SACN.Enabled)
		}
	}

	if cfg.ArtNet.Target != "" {
		a, err := artnet.New(
			cfg.ArtNet.Target,
			cfg.ArtNet.Universe,
			cfg.ArtNet.Channels.Beat,
			cfg.ArtNet.Channels.Downbeat,
			cfg.ArtNet.Channels.Slow,
			cfg.ArtNet.PulseMS,
		)
		if err != nil {
			logger.Warn("Art-Net sender init failed", "error", err)
		} else {
			m.artSender = a
			if cfg.ArtNet.Enabled {
				atomic.StoreInt32(&m.artOn, 1)
			}
			logger.Info("Art-Net sender ready",
				"universe", cfg.ArtNet.Universe,
				"enabled", cfg.ArtNet.Enabled)
		}
	}

	if cfg.OSC.Target != "" {
		o, err := osc.New(
			cfg.OSC.Target,
			cfg.OSC.Addresses.Beat,
			cfg.OSC.Addresses.Downbeat,
			cfg.OSC.Addresses.Slow,
			cfg.OSC.PulseMS,
		)
		if err != nil {
			logger.Warn("OSC sender init failed", "error", err)
		} else {
			m.oscSender = o
			if cfg.OSC.Enabled {
				atomic.StoreInt32(&m.oscOn, 1)
			}
			logger.Info("OSC sender ready",
				"target", cfg.OSC.Target,
				"enabled", cfg.OSC.Enabled)
		}
	}

	return m
}

// Dispatch is called by the Tracker on each beat event.
// event: "beat" | "downbeat" | "slow"
func (m *Manager) Dispatch(event string, _ int) {
	switch event {
	case "beat":
		if m.sacnSender != nil && atomic.LoadInt32(&m.sacnOn) == 1 {
			m.sacnSender.Beat()
		}
		if m.artSender != nil && atomic.LoadInt32(&m.artOn) == 1 {
			m.artSender.Beat()
		}
		if m.oscSender != nil && atomic.LoadInt32(&m.oscOn) == 1 {
			m.oscSender.Beat()
		}

	case "downbeat":
		if m.sacnSender != nil && atomic.LoadInt32(&m.sacnOn) == 1 {
			m.sacnSender.Downbeat()
		}
		if m.artSender != nil && atomic.LoadInt32(&m.artOn) == 1 {
			m.artSender.Downbeat()
		}
		if m.oscSender != nil && atomic.LoadInt32(&m.oscOn) == 1 {
			m.oscSender.Downbeat()
		}

	case "slow":
		if m.sacnSender != nil && atomic.LoadInt32(&m.sacnOn) == 1 {
			m.sacnSender.Slow()
		}
		if m.artSender != nil && atomic.LoadInt32(&m.artOn) == 1 {
			m.artSender.Slow()
		}
		if m.oscSender != nil && atomic.LoadInt32(&m.oscOn) == 1 {
			m.oscSender.Slow()
		}
	}
}

// SetEnabled toggles a protocol sender at runtime.
func (m *Manager) SetEnabled(protocol string, enabled bool) {
	v := int32(0)
	if enabled {
		v = 1
	}
	switch protocol {
	case "sacn":
		atomic.StoreInt32(&m.sacnOn, v)
	case "artnet":
		atomic.StoreInt32(&m.artOn, v)
	case "osc":
		atomic.StoreInt32(&m.oscOn, v)
	}
}

// Enabled returns the current runtime enable state of all three protocols.
func (m *Manager) Enabled() map[string]bool {
	return map[string]bool{
		"sacn":   atomic.LoadInt32(&m.sacnOn) == 1,
		"artnet": atomic.LoadInt32(&m.artOn) == 1,
		"osc":    atomic.LoadInt32(&m.oscOn) == 1,
	}
}

// Close shuts down all active senders.
func (m *Manager) Close() {
	if m.sacnSender != nil {
		m.sacnSender.Close()
	}
	if m.artSender != nil {
		m.artSender.Close()
	}
	if m.oscSender != nil {
		m.oscSender.Close()
	}
}
