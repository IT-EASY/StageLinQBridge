package mainconn

import (
	"bufio"
	"context"
	"io"
	"net"
	"sync"

	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

// PeerEvent is emitted once per unique remote IP when a device sends a services request.
type PeerEvent struct {
	RemoteIP  net.IP
	PeerToken token.Token
}

type Server struct {
	logger        *debug.Logger
	listener      net.Listener
	clientToken   token.Token
	stateMapPort  uint16
	beatInfoPort  uint16

	peerEvents chan PeerEvent
	seenIPs    map[string]bool
	seenMu     sync.Mutex
}

func NewServer(logger *debug.Logger, clientToken token.Token, stateMapPort uint16, beatInfoPort uint16) (*Server, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	return &Server{
		logger:       logger,
		listener:     ln,
		clientToken:  clientToken,
		stateMapPort: stateMapPort,
		beatInfoPort: beatInfoPort,
		peerEvents:   make(chan PeerEvent, 8),
		seenIPs:      make(map[string]bool),
	}, nil
}

func (s *Server) Port() uint16 {
	return uint16(s.listener.Addr().(*net.TCPAddr).Port)
}

// PeerConnected returns a channel that emits once per unique device IP when a
// services request is received. Callers use this to initiate an outbound
// connection back to the device (go-stagelinq handshake pattern).
func (s *Server) PeerConnected() <-chan PeerEvent {
	return s.peerEvents
}

func (s *Server) Serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.listener.Close()
		close(s.peerEvents)
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.logger.Warn("main server accept failed", "error", err)
			return
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP

	s.logger.Info("incoming main connection", "remote", remote)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	br := bufio.NewReader(conn)

	for {
		messageID, message, err := ParseMessage(br)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err == io.EOF {
				s.logger.Debug("incoming main connection closed", "remote", remote)
				return
			}
			s.logger.Warn("incoming main connection read failed", "remote", remote, "error", err)
			return
		}

		switch typed := message.(type) {
		case token.Token:
			s.logger.Debug("incoming services request", "remote", remote, "token", typed.Hex())

			// Announce our own services.
			ann := BuildServiceAnnouncement(s.clientToken, "StateMap", s.stateMapPort)
			if _, err := conn.Write(ann); err != nil {
				s.logger.Warn("service announcement failed", "error", err)
				return
			}
			if s.beatInfoPort != 0 {
				ann2 := BuildServiceAnnouncement(s.clientToken, "BeatInfo", s.beatInfoPort)
				if _, err := conn.Write(ann2); err != nil {
					s.logger.Warn("beatinfo announcement failed", "error", err)
					return
				}
			}

			// Ask for the device's services.
			req := BuildServicesRequest(s.clientToken)
			if _, err := conn.Write(req); err != nil {
				s.logger.Warn("services reply failed", "error", err)
				return
			}

			// Notify once per device IP so the caller can initiate an outbound
			// connection after we've been recognised as a peer.
			s.seenMu.Lock()
			ipStr := remoteIP.String()
			first := !s.seenIPs[ipStr]
			if first {
				s.seenIPs[ipStr] = true
			}
			s.seenMu.Unlock()

			if first {
				select {
				case s.peerEvents <- PeerEvent{RemoteIP: remoteIP, PeerToken: typed}:
				default:
				}
			}

		case ServiceAnnouncement:
			s.logger.Info(
				"service announced (incoming)",
				"remote", remote,
				"name", typed.Name,
				"port", typed.Port,
			)

		case TimeStamp:
			s.logger.Trace("timestamp received (incoming)", "remote", remote, "timeAlive", typed.TimeAlive)

		default:
			s.logger.Trace("incoming message", "remote", remote, "id", messageID)
		}
	}
}
