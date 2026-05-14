package beatinfo

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// beatEmitMagic is the subtype marker for beat emit messages.
var beatEmitMagic = []byte{0x00, 0x00, 0x00, 0x02}

// beatStartMagic is the payload for the start-stream command.
var beatStartMagic = []byte{0x00, 0x00, 0x00, 0x00}

// PlayerInfo contains per-deck beat information for a single beat emit.
type PlayerInfo struct {
	Beat       float64
	TotalBeats float64
	BPM        float64
}

// BeatEvent is a single beat emit message from a StagelinQ device.
type BeatEvent struct {
	Clock     uint64
	Players   []PlayerInfo
	Timelines []float64
}

// WriteStartStream sends the start-stream command to the given writer.
// Format: uint32(4) + [00 00 00 00]  (length-prefixed, 8 bytes total).
func WriteStartStream(w io.Writer) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], 4)
	copy(buf[4:8], beatStartMagic)
	_, err := w.Write(buf)
	return err
}

// ReadBeatEvent reads one beat emit message from r.
// Returns nil, nil for non-emit messages (e.g. unknown subtypes).
func ReadBeatEvent(r io.Reader) (*BeatEvent, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	if len(payload) < 4 {
		return nil, errors.New("beatinfo: payload too short")
	}

	// check magic
	if payload[0] != beatEmitMagic[0] || payload[1] != beatEmitMagic[1] ||
		payload[2] != beatEmitMagic[2] || payload[3] != beatEmitMagic[3] {
		// Unknown subtype — skip gracefully.
		return nil, nil
	}

	if len(payload) < 4+8+4 {
		return nil, errors.New("beatinfo: payload too short for beat emit header")
	}

	offset := 4
	clock := binary.BigEndian.Uint64(payload[offset:])
	offset += 8

	numRecords := int(binary.BigEndian.Uint32(payload[offset:]))
	offset += 4

	// Each player record is 3 × float64 = 24 bytes.
	if len(payload) < offset+numRecords*24+numRecords*8 {
		return nil, errors.New("beatinfo: payload too short for player records")
	}

	players := make([]PlayerInfo, numRecords)
	for i := range players {
		players[i].Beat = readFloat64(payload[offset:])
		offset += 8
		players[i].TotalBeats = readFloat64(payload[offset:])
		offset += 8
		players[i].BPM = readFloat64(payload[offset:])
		offset += 8
	}

	timelines := make([]float64, numRecords)
	for i := range timelines {
		timelines[i] = readFloat64(payload[offset:])
		offset += 8
	}

	return &BeatEvent{
		Clock:     clock,
		Players:   players,
		Timelines: timelines,
	}, nil
}

// readFloat64 reads a big-endian float64 from b.
func readFloat64(b []byte) float64 {
	return math.Float64frombits(binary.BigEndian.Uint64(b))
}
