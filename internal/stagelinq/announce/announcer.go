package announce

import (
	"context"
	"net"
	"time"

	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/discovery"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

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

	target *net.UDPAddr

	source          string
	softwareName    string
	softwareVersion string
	token           token.Token

	interval time.Duration
}

func New(logger *debug.Logger) *Announcer {
	return &Announcer{
		logger: logger,
		target: &net.UDPAddr{
			IP:   net.IPv4bcast,
			Port: discovery.DefaultPort,
		},
		source:          DefaultSource,
		softwareName:    DefaultSoftwareName,
		softwareVersion: DefaultSoftwareVersion,
		token:           token.SoundSwitch,
		interval:        DefaultInterval,
	}
}

func (a *Announcer) Start(ctx context.Context) error {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	})
	if err != nil {
		return err
	}

	go func() {
		defer conn.Close()

		a.send(conn, ActionHowdy)

		ticker := time.NewTicker(a.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				a.send(conn, ActionExit)
				return

			case <-ticker.C:
				a.send(conn, ActionHowdy)
			}
		}
	}()

	return nil
}

func (a *Announcer) send(conn *net.UDPConn, action string) {
	packet := discovery.Packet{
		Token:           a.token,
		Source:          a.source,
		Action:          action,
		SoftwareName:    a.softwareName,
		SoftwareVersion: a.softwareVersion,
		Port:            0,
	}

	data := discovery.BuildPacket(packet)

	_, err := conn.WriteToUDP(data, a.target)
	if err != nil {
		a.logger.Warn("announce failed", "error", err)
		return
	}

	a.logger.Trace(
		"announce sent",
		"action", action,
		"target", a.target.String(),
		"token", a.token.Hex(),
	)
}
