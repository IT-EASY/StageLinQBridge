package beatinfo

import (
	"bufio"
	"context"
	"io"
	"net"

	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/mainconn"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

// Server listens for incoming BeatInfo connections from StagelinQ devices.
// Protocol (by symmetry with StateMap):
//  1. Device connects and sends hello (serviceAnnouncementMessage).
//  2. We send our hello back.
//  3. We send StartStream.
//  4. Device streams beatEmitMessages.
type Server struct {
	logger      *debug.Logger
	listener    net.Listener
	clientToken token.Token

	beats chan BeatEvent
}

// NewServer creates a BeatInfo TCP server.
func NewServer(logger *debug.Logger, clientToken token.Token) (*Server, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	return &Server{
		logger:      logger,
		listener:    ln,
		clientToken: clientToken,
		beats:       make(chan BeatEvent, 64),
	}, nil
}

func (s *Server) Port() uint16 {
	return uint16(s.listener.Addr().(*net.TCPAddr).Port)
}

// Beats returns a channel that emits beat events as they arrive.
func (s *Server) Beats() <-chan BeatEvent {
	return s.beats
}

func (s *Server) Serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.listener.Close()
		close(s.beats)
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.logger.Warn("beatinfo server accept failed", "error", err)
			return
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()

	s.logger.Info("incoming BeatInfo connection", "remote", remote)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	br := bufio.NewReader(conn)

	// Step 1: read device's hello (same format as StateMap hello).
	_, msg, err := mainconn.ParseMessage(br)
	if err != nil {
		if err != io.EOF && ctx.Err() == nil {
			s.logger.Warn("BeatInfo hello read failed", "remote", remote, "error", err)
		}
		return
	}

	hello, ok := msg.(mainconn.ServiceAnnouncement)
	if !ok {
		s.logger.Warn("unexpected first message on BeatInfo port", "remote", remote)
		return
	}

	s.logger.Info("BeatInfo hello received",
		"remote", remote,
		"service", hello.Name,
		"token", hello.Token.Hex(),
	)

	// Step 2: send StartStream — no hello-back needed (PRIME 4 is the client,
	// it doesn't expect a service announcement reply from us here).
	if err := WriteStartStream(conn); err != nil {
		s.logger.Warn("BeatInfo StartStream failed", "remote", remote, "error", err)
		return
	}
	s.logger.Debug("BeatInfo StartStream sent", "remote", remote)

	// Step 4: read beat emit messages until connection closes.
	for {
		evt, err := ReadBeatEvent(br)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err == io.EOF {
				s.logger.Debug("BeatInfo connection closed", "remote", remote)
				return
			}
			s.logger.Warn("BeatInfo read failed", "remote", remote, "error", err)
			return
		}
		if evt == nil {
			continue // non-emit message
		}

		s.logger.Debug("BeatInfo event received",
			"remote", remote,
			"clock", evt.Clock,
			"players", len(evt.Players),
		)

		select {
		case s.beats <- *evt:
		default:
		}
	}
}
