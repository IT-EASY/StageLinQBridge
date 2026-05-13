package statemap

import (
	"bufio"
	"context"
	"io"
	"net"

	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/mainconn"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

// StateUpdate is emitted whenever a subscribed state value changes.
type StateUpdate struct {
	RemoteIP net.IP
	Name     string
	Value    string // raw JSON
}

// Server listens for incoming StateMap connections from StagelinQ devices.
// When a device connects it exchanges a hello handshake, subscribes to the
// given state names, and emits received values on StateUpdates().
type Server struct {
	logger      *debug.Logger
	listener    net.Listener
	clientToken token.Token
	subscriptions []string

	updates chan StateUpdate
}

// NewServer creates a StateMap TCP server that will subscribe to the given
// state paths once a device connects.
func NewServer(logger *debug.Logger, clientToken token.Token, subscriptions []string) (*Server, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	return &Server{
		logger:        logger,
		listener:      ln,
		clientToken:   clientToken,
		subscriptions: subscriptions,
		updates:       make(chan StateUpdate, 64),
	}, nil
}

func (s *Server) Port() uint16 {
	return uint16(s.listener.Addr().(*net.TCPAddr).Port)
}

// StateUpdates returns a channel that emits state values as they arrive.
func (s *Server) StateUpdates() <-chan StateUpdate {
	return s.updates
}

func (s *Server) Serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.listener.Close()
		close(s.updates)
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.logger.Warn("statemap server accept failed", "error", err)
			return
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP

	s.logger.Info("incoming StateMap connection", "remote", remote)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	br := bufio.NewReader(conn)

	// Step 1: read device's hello (main-connection format serviceAnnouncementMessage).
	_, msg, err := mainconn.ParseMessage(br)
	if err != nil {
		if err != io.EOF && ctx.Err() == nil {
			s.logger.Warn("StateMap hello read failed", "remote", remote, "error", err)
		}
		return
	}

	hello, ok := msg.(mainconn.ServiceAnnouncement)
	if !ok {
		s.logger.Warn("unexpected first message on StateMap port", "remote", remote)
		return
	}

	s.logger.Info("StateMap hello received",
		"remote", remote,
		"service", hello.Name,
		"token", hello.Token.Hex(),
	)

	// Step 2: send our hello back — our local source port as the "port" field.
	ourPort := uint16(conn.LocalAddr().(*net.TCPAddr).Port)
	helloReply := mainconn.BuildServiceAnnouncement(s.clientToken, "StateMap", ourPort)
	if _, err := conn.Write(helloReply); err != nil {
		s.logger.Warn("StateMap hello reply failed", "remote", remote, "error", err)
		return
	}

	// Step 3: send subscriptions for each configured state path.
	for _, path := range s.subscriptions {
		if err := WriteSubscribe(conn, path, 0); err != nil {
			s.logger.Warn("StateMap subscribe failed", "remote", remote, "path", path, "error", err)
			return
		}
	}
	s.logger.Debug("StateMap subscriptions sent", "remote", remote, "count", len(s.subscriptions))

	// Step 4: read state emit messages until connection closes.
	for {
		evt, err := ReadStateEvent(br)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err == io.EOF {
				s.logger.Debug("StateMap connection closed", "remote", remote)
				return
			}
			s.logger.Warn("StateMap read failed", "remote", remote, "error", err)
			return
		}
		if evt == nil {
			continue // non-emit message (ack, etc.)
		}

		s.logger.Debug("StateMap state received",
			"remote", remote,
			"name", evt.Name,
			"value", evt.Value,
		)

		select {
		case s.updates <- StateUpdate{RemoteIP: remoteIP, Name: evt.Name, Value: evt.Value}:
		default:
		}
	}
}
