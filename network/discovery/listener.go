package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/it-easy/stagelinq-go/internal/debug"
)

type Listener struct {
	logger debug.Logger
	conn   *net.UDPConn
}

func NewListener(logger debug.Logger) *Listener {
	return &Listener{
		logger: logger,
	}
}

func (l *Listener) Listen(ctx context.Context) (<-chan *Device, error) {
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: DefaultPort,
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}

	l.conn = conn

	devices := make(chan *Device)

	go func() {
		defer close(devices)
		defer conn.Close()

		buffer := make([]byte, MaxPacketSize)

		l.logger.Info("discovery listener started", "port", DefaultPort)

		for {
			select {
			case <-ctx.Done():
				l.logger.Info("discovery listener stopped")
				return

			default:
			}

			_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			packet := make([]byte, n)
			copy(packet, buffer[:n])

			device, err := ParsePacket(packet)
			if err != nil {
				l.logger.Debug("invalid discovery packet", "error", err)
				continue
			}

			device.IP = remoteAddr.IP
			device.LastSeen = time.Now()

			devices <- device
		}
	}()

	return devices, nil
}
