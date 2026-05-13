package mainconn

import (
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
	logger *debug.Logger
	device discovery.Device

	conn net.Conn

	mutex    sync.RWMutex
	services map[string]uint16
}

func NewClient(logger *debug.Logger, device discovery.Device) *Client {
	return &Client{
		logger:   logger,
		device:   device,
		services: make(map[string]uint16),
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

	err = c.SendServicesRequest()
	if err != nil {
		_ = conn.Close()
		return err
	}

	go c.readLoop(ctx)

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

	data := BuildServicesRequest(token.SoundSwitch)

	_, err := c.conn.Write(data)
	if err != nil {
		return err
	}

	c.logger.Debug("services request sent", "token", token.SoundSwitch.Hex())

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

func (c *Client) readLoop(ctx context.Context) {
	defer c.logger.Debug("main connection read loop stopped")

	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			_ = c.Close()
			return
		default:
		}

		_ = c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		count, err := c.conn.Read(buffer)
		if err != nil {
			if isTimeout(err) {
				continue
			}

			if err == io.EOF {
				c.logger.Warn("main connection closed by remote")
				return
			}

			c.logger.Warn("main connection read failed", "error", err)
			return
		}

		messageID, message, err := ParseMessage(buffer[:count])
		if err != nil {
			c.logger.Trace("ignored invalid main connection message", "error", err)
			continue
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
			c.logger.Debug(
				"services request received",
				"token", typed.Hex(),
			)

		default:
			c.logger.Trace(
				"main message received",
				"id", messageID,
			)
		}
	}
}

func isTimeout(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}
