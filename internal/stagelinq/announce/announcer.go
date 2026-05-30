package announce

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/network"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/discovery"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

// lanIP may be nil — in that case all active LAN interfaces are used.

const (
	DefaultInterval = 1 * time.Second

	ActionHowdy = "DISCOVERER_HOWDY_"
	ActionExit  = "DISCOVERER_EXIT__"

	DefaultSource          = "StageLinQBridge"
	DefaultSoftwareName    = "StageLinQBridge"
	DefaultSoftwareVersion = "0.1.0"
)

type Announcer struct {
	logger *debug.Logger

	lanIP           net.IP
	port            uint16
	source          string
	softwareName    string
	softwareVersion string
	clientToken     token.Token
	mu              sync.Mutex // protects clientToken

	interval time.Duration
}

func New(logger *debug.Logger, clientToken token.Token, lanIP net.IP, port uint16) *Announcer {
	return &Announcer{
		logger:          logger,
		lanIP:           lanIP,
		port:            port,
		source:          DefaultSource,
		softwareName:    DefaultSoftwareName,
		softwareVersion: DefaultSoftwareVersion,
		clientToken:     clientToken,
		interval:        DefaultInterval,
	}
}

func (a *Announcer) Start(ctx context.Context) error {
	// Send EXIT first so any device that still has our IP in its peer table
	// removes the stale entry before we announce as a fresh peer.
	a.send(ActionExit)
	time.Sleep(300 * time.Millisecond)

	a.send(ActionHowdy)

	ticker := time.NewTicker(a.interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				a.send(ActionExit)
				return

			case <-ticker.C:
				a.send(ActionHowdy)
			}
		}
	}()

	return nil
}

func (a *Announcer) ClientToken() token.Token {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.clientToken
}

// RotateToken sends EXIT with the current token, switches to newToken, and
// immediately sends HOWDY. The background ticker continues with the new token.
func (a *Announcer) RotateToken(newToken token.Token) {
	a.send(ActionExit)
	a.mu.Lock()
	a.clientToken = newToken
	a.mu.Unlock()
	time.Sleep(200 * time.Millisecond)
	a.send(ActionHowdy)
}

func (a *Announcer) IsOwnDevice(device discovery.Device) bool {
	return device.Token == a.clientToken &&
		device.Source == a.source &&
		device.SoftwareName == a.softwareName &&
		device.SoftwareVersion == a.softwareVersion
}

func (a *Announcer) send(action string) {
	data := discovery.BuildPacket(discovery.Packet{
		Token:           a.clientToken,
		Source:          a.source,
		Action:          action,
		SoftwareName:    a.softwareName,
		SoftwareVersion: a.softwareVersion,
		Port:            a.port,
	})

	broadcastIPs, err := network.BroadcastIPs(a.lanIP)
	if err != nil {
		a.logger.Warn("failed to get broadcast IPs", "error", err)
		return
	}

	for _, ip := range broadcastIPs {
		target := &net.UDPAddr{IP: ip, Port: discovery.DefaultPort}

		// Bind to lanIP so the outgoing UDP packet carries the correct source
		// address. Without this, Windows picks arbitrarily on multi-homed
		// adapters — the PRIME 4 would then try to connect back to the wrong IP.
		var localAddr *net.UDPAddr
		if a.lanIP != nil {
			localAddr = &net.UDPAddr{IP: a.lanIP, Port: 0}
		}

		conn, err := net.DialUDP("udp4", localAddr, target)
		if err != nil {
			a.logger.Warn("announce dial failed", "target", target.String(), "error", err)
			continue
		}
		_, err = conn.Write(data)
		conn.Close()
		if err != nil {
			a.logger.Warn("announce write failed", "target", target.String(), "error", err)
			continue
		}
		a.logger.Trace("announce sent", "action", action, "target", target.String(), "token", a.clientToken.Hex())
	}
}
