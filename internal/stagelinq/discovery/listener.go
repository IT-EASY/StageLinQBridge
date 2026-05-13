package discovery

import (
	"context"
	"encoding/hex"
	"net"
	"time"

	"github.com/it-easy/StageLinQBridge/internal/debug"
)

const DefaultPort = 51337

type Listener struct {
	logger *debug.Logger
	port   int
}

func NewListener(logger *debug.Logger) *Listener {
	return &Listener{
		logger: logger,
		port:   DefaultPort,
	}
}

func NewListenerWithPort(logger *debug.Logger, port int) *Listener {
	return &Listener{
		logger: logger,
		port:   port,
	}
}

func (l *Listener) Listen(ctx context.Context) (<-chan Device, error) {
	address := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: l.port,
	}

	conn, err := net.ListenUDP("udp4", address)
	if err != nil {
		return nil, err
	}

	devices := make(chan Device)

	go func() {
		defer close(devices)
		defer conn.Close()

		buffer := make([]byte, 4096)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

			count, remote, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if isTimeout(err) {
					continue
				}

				l.logger.Warn("discovery read failed", "error", err)
				continue
			}

			packet, err := ParsePacket(buffer[:count])
			if err != nil {
				l.logger.Debug("ignored invalid discovery packet",
					"src", remote.String(),
					"error", err,
					"bytes", hex.EncodeToString(buffer[:count]),
				)
				continue
			}

			devices <- Device{
				Source:          packet.Source,
				Action:          packet.Action,
				SoftwareName:    packet.SoftwareName,
				SoftwareVersion: packet.SoftwareVersion,
				IP:              remote.IP,
				Port:            packet.Port,
				Token:           packet.Token,
			}
		}
	}()

	return devices, nil
}

func isTimeout(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}
