// Package osc implements a minimal OSC 1.0 UDP sender.
// Sends float32 value 1.0 on trigger, then 0.0 after pulse_ms.
package osc

import (
	"encoding/binary"
	"math"
	"net"
	"time"
)

// Sender sends OSC messages via UDP unicast.
type Sender struct {
	conn     *net.UDPConn
	dst      *net.UDPAddr
	pulseMS  int
	beatAddr []byte
	dbAddr   []byte
	slowAddr []byte
}

// New resolves the target address and creates a UDP sender.
func New(target string, beatAddr, downbeatAddr, slowAddr string, pulseMS int) (*Sender, error) {
	dst, err := net.ResolveUDPAddr("udp4", target)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	return &Sender{
		conn:     conn,
		dst:      dst,
		pulseMS:  pulseMS,
		beatAddr: oscPad(beatAddr),
		dbAddr:   oscPad(downbeatAddr),
		slowAddr: oscPad(slowAddr),
	}, nil
}

func (s *Sender) Beat()     { go s.flash(s.beatAddr) }
func (s *Sender) Downbeat() { go s.flash(s.dbAddr) }
func (s *Sender) Slow()     { go s.flash(s.slowAddr) }

func (s *Sender) Close() { s.conn.Close() }

func (s *Sender) flash(addr []byte) {
	_, _ = s.conn.WriteTo(buildMsg(addr, 1.0), s.dst)
	time.Sleep(time.Duration(s.pulseMS) * time.Millisecond)
	_, _ = s.conn.WriteTo(buildMsg(addr, 0.0), s.dst)
}

// buildMsg constructs an OSC message: <addr>,f<pad><float32>.
func buildMsg(addr []byte, val float32) []byte {
	tag := oscPad(",f")
	fval := make([]byte, 4)
	binary.BigEndian.PutUint32(fval, math.Float32bits(val))
	msg := make([]byte, len(addr)+len(tag)+4)
	copy(msg, addr)
	copy(msg[len(addr):], tag)
	copy(msg[len(addr)+len(tag):], fval)
	return msg
}

// oscPad null-terminates a string and pads to the next 4-byte boundary.
func oscPad(s string) []byte {
	b := append([]byte(s), 0) // null-terminate
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}
