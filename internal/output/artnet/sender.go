// Package artnet implements an Art-Net ArtDmx sender (unicast only).
//
// Broadcast is intentionally not supported — beat triggers fire continuously
// and would flood the entire network segment. A target IP must be configured.
package artnet

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	defaultPort = 6454
	maxChannels = 512
)

var artNetID = [8]byte{'A', 'r', 't', '-', 'N', 'e', 't', 0x00}

// Sender sends Art-Net ArtDmx packets via UDP unicast.
type Sender struct {
	mu       sync.Mutex
	conn     *net.UDPConn
	dst      *net.UDPAddr
	universe uint16
	dmx      [maxChannels]byte
	seq      uint8
	pulseMS  int
	beatCh   uint16
	downbeat uint16
	slow     uint16
}

// New resolves target (accepts "ip" or "ip:port") and creates a unicast sender.
func New(target string, universe uint16, beatCh, downbeat, slow uint16, pulseMS int) (*Sender, error) {
	if target == "" {
		return nil, fmt.Errorf("art-net target IP must be configured (no broadcast)")
	}

	// Append default port if missing.
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		// Assume bare IP, append default port.
		host = target
		port = fmt.Sprintf("%d", defaultPort)
	}
	dst, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("art-net target: %w", err)
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}

	return &Sender{
		conn:     conn,
		dst:      dst,
		universe: universe,
		pulseMS:  pulseMS,
		beatCh:   beatCh,
		downbeat: downbeat,
		slow:     slow,
	}, nil
}

func (s *Sender) Beat()     { go s.flash(s.beatCh) }
func (s *Sender) Downbeat() { go s.flash(s.downbeat) }
func (s *Sender) Slow()     { go s.flash(s.slow) }

func (s *Sender) Close() { s.conn.Close() }

func (s *Sender) flash(ch uint16) {
	s.set(ch, 255)
	time.Sleep(time.Duration(s.pulseMS) * time.Millisecond)
	s.set(ch, 0)
}

func (s *Sender) set(ch uint16, val byte) {
	if ch < 1 || ch > maxChannels {
		return
	}
	s.mu.Lock()
	s.dmx[ch-1] = val
	pkt := s.buildPacket()
	s.seq++
	s.mu.Unlock()
	_, _ = s.conn.WriteTo(pkt, s.dst)
}

// buildPacket constructs an ArtDmx packet for 512 channels (530 bytes).
// Called with s.mu held.
func (s *Sender) buildPacket() []byte {
	buf := make([]byte, 18+maxChannels)
	copy(buf[0:], artNetID[:])
	binary.LittleEndian.PutUint16(buf[8:], 0x5000)      // OpCode: OpDmx
	binary.BigEndian.PutUint16(buf[10:], 14)             // ProtVer
	buf[12] = s.seq                                      // Sequence
	// buf[13] = 0 Physical
	binary.LittleEndian.PutUint16(buf[14:], s.universe)  // Universe (port-address)
	binary.BigEndian.PutUint16(buf[16:], maxChannels)    // Length
	copy(buf[18:], s.dmx[:])
	return buf
}
