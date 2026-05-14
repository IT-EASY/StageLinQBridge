// Package sacn implements an E1.31 sACN sender (multicast, no unicast).
// One packet is sent per universe per trigger; only the three beat channels
// are modified — all other channels remain at 0.
package sacn

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

const (
	udpPort     = 5568
	maxChannels = 512
	priority    = 100
)

// acnPacketID is the fixed ACN Packet Identifier required by E1.31 §6.2.
var acnPacketID = [12]byte{
	0x41, 0x53, 0x43, 0x2d, 0x45, 0x31, 0x2e,
	0x31, 0x37, 0x00, 0x00, 0x00,
}

// Sender sends E1.31 sACN packets via multicast.
type Sender struct {
	mu       sync.Mutex
	conn     *net.UDPConn
	dst      *net.UDPAddr
	universe uint16
	dmx      [maxChannels]byte
	seq      uint8
	cid      [16]byte
	pulseMS  int
	beatCh   uint16
	downbeat uint16
	slow     uint16
}

// New creates and binds a sACN sender for the given universe.
func New(universe uint16, beatCh, downbeat, slow uint16, pulseMS int) (*Sender, error) {
	var cid [16]byte
	if _, err := rand.Read(cid[:]); err != nil {
		return nil, err
	}
	// Set version bits per RFC 4122 §4.4 (random UUID).
	cid[6] = (cid[6] & 0x0f) | 0x40
	cid[8] = (cid[8] & 0x3f) | 0x80

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}

	hi := byte(universe >> 8)
	lo := byte(universe & 0xff)
	dst := &net.UDPAddr{
		IP:   net.IPv4(239, 255, hi, lo),
		Port: udpPort,
	}

	return &Sender{
		conn:     conn,
		dst:      dst,
		universe: universe,
		cid:      cid,
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

// buildPacket constructs a full E1.31 packet (638 bytes) for 512 channels.
// Called with s.mu held.
func (s *Sender) buildPacket() []byte {
	const totalLen = 638
	buf := make([]byte, totalLen)

	// ── Root Layer ───────────────────────────────────────────────────────────
	binary.BigEndian.PutUint16(buf[0:], 0x0010) // Preamble size
	binary.BigEndian.PutUint16(buf[2:], 0x0000) // Postamble size
	copy(buf[4:], acnPacketID[:])               // ACN Packet Identifier
	// Flags & Length: flags=0x70, length = totalLen-16
	binary.BigEndian.PutUint16(buf[16:], 0x7000|uint16(totalLen-16))
	binary.BigEndian.PutUint32(buf[18:], 0x00000004) // Vector: VECTOR_ROOT_E131_DATA
	copy(buf[22:], s.cid[:])                          // CID

	// ── Framing Layer ────────────────────────────────────────────────────────
	binary.BigEndian.PutUint16(buf[38:], 0x7000|uint16(totalLen-38))
	binary.BigEndian.PutUint32(buf[40:], 0x00000002) // Vector: VECTOR_E131_DATA_PACKET
	copy(buf[44:], "StageLinQBridge")                // Source Name (64 bytes, rest=0)
	buf[108] = priority                              // Priority
	// buf[109:111] = 0x0000 (Synchronization Address — not used)
	buf[111] = s.seq  // Sequence Number
	// buf[112] = 0x00 (Options)
	binary.BigEndian.PutUint16(buf[113:], s.universe) // Universe

	// ── DMP Layer ────────────────────────────────────────────────────────────
	binary.BigEndian.PutUint16(buf[115:], 0x7000|uint16(totalLen-115))
	buf[117] = 0x02 // Vector: VECTOR_DMP_SET_PROPERTY
	buf[118] = 0xa1 // Address Type & Data Type
	// buf[119:121] = 0x0000 (First Property Address)
	binary.BigEndian.PutUint16(buf[121:], 0x0001) // Address Increment
	binary.BigEndian.PutUint16(buf[123:], 0x0201) // Property count: 513 (start code + 512 ch)
	// buf[125] = 0x00 (DMX start code)
	copy(buf[126:], s.dmx[:]) // 512 channel values

	return buf
}
