package mainconn

import (
	"bufio"
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/discovery"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

type Client struct {
	logger       *debug.Logger
	device       discovery.Device
	clientToken  token.Token
	stateMapPort uint16

	conn net.Conn

	mutex    sync.RWMutex
	services map[string]uint16
}

func NewClient(logger *debug.Logger, device discovery.Device, clientToken token.Token, stateMapPort uint16) *Client {
	return &Client{
		logger:       logger,
		device:       device,
		clientToken:  clientToken,
		stateMapPort: stateMapPort,
		services:     make(map[string]uint16),
	}
}

func (c *Client) Connect(ctx context.Context) error {
	conn, err := net.DialTimeout("tcp4", c.device.Address(), 5*time.Second)
	if err != nil {
		return err
	}

	c.conn = conn

	c.logger.Info(
		"main connection established",
		"address", c.device.Address(),
		"source", c.device.Source,
		"software", c.device.SoftwareName,
		"version", c.device.SoftwareVersion,
	)

	// Start keepalive and read loop before sending anything — matches the
	// reference implementation (go-stagelinq) which never sends type-0
	// proactively on outbound; it only announces services in response to a
	// type-2 ServicesRequest received from the remote side.
	go c.keepaliveLoop(ctx)
	go c.readLoop(ctx)

	req := BuildServicesRequest(c.clientToken)
	if _, err := conn.Write(req); err != nil {
		_ = conn.Close()
		return err
	}

	return nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *Client) SendServicesRequest() error {
	if c.conn == nil {
		return net.ErrClosed
	}

	data := BuildServicesRequest(c.clientToken)

	_, err := c.conn.Write(data)
	if err != nil {
		return err
	}

	c.logger.Debug("services request sent", "token", c.clientToken.Hex())

	return nil
}

func (c *Client) Services() map[string]uint16 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make(map[string]uint16, len(c.services))
	for name, port := range c.services {
		result[name] = port
	}

	return result
}

func (c *Client) ServicePort(name string) (uint16, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	port, ok := c.services[name]
	return port, ok
}

func (c *Client) WaitForService(ctx context.Context, name string) (uint16, bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if port, ok := c.ServicePort(name); ok {
			return port, true
		}

		select {
		case <-ctx.Done():
			return 0, false

		case <-ticker.C:
		}
	}
}

func (c *Client) keepaliveLoop(ctx context.Context) {
	defer c.logger.Debug("main connection keepalive loop stopped")

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			data := BuildReferenceMessage(c.clientToken, c.device.Token)
			if _, err := c.conn.Write(data); err != nil {
				c.logger.Warn("keepalive send failed", "error", err)
				return
			}
		}
	}
}

func (c *Client) readLoop(ctx context.Context) {
	defer c.logger.Debug("main connection read loop stopped")

	// Close the connection when the context is cancelled so the blocking
	// ParseMessage call below returns immediately.
	go func() {
		<-ctx.Done()
		_ = c.conn.Close()
	}()

	br := bufio.NewReader(c.conn)

	for {
		messageID, message, err := ParseMessage(br)
		if err != nil {
			if ctx.Err() != nil {
				return // normal shutdown
			}
			if err == io.EOF {
				c.logger.Warn("main connection closed by remote")
				return
			}
			c.logger.Warn("main connection read failed", "error", err)
			return
		}

		switch typed := message.(type) {
		case ServiceAnnouncement:
			c.mutex.Lock()
			c.services[typed.Name] = typed.Port
			c.mutex.Unlock()

			c.logger.Info(
				"service announced",
				"name", typed.Name,
				"port", typed.Port,
			)

		case TimeStamp:
			c.logger.Trace(
				"timestamp received",
				"timeAlive", typed.TimeAlive,
			)

		case token.Token:
			// PRIME 4 is asking what services we offer — respond with our StateMap port.
			c.logger.Debug(
				"services request received — announcing our StateMap",
				"token", typed.Hex(),
			)
			ann := BuildServiceAnnouncement(c.clientToken, "StateMap", c.stateMapPort)
			if _, err := c.conn.Write(ann); err != nil {
				c.logger.Warn("failed to announce StateMap in response to services request", "error", err)
			}

		default:
			c.logger.Trace(
				"main message received",
				"id", messageID,
			)
		}
	}
}
