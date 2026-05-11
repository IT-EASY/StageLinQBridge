package discovery

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

type packetReader struct {
	data []byte
	pos  int
}

func newPacketReader(data []byte) *packetReader {
	return &packetReader{data: data}
}

func (r *packetReader) remaining() int {
	return len(r.data) - r.pos
}

func (r *packetReader) readBytes(n int) ([]byte, error) {
	if n < 0 || r.remaining() < n {
		return nil, fmt.Errorf("need %d bytes, have %d", n, r.remaining())
	}

	out := r.data[r.pos : r.pos+n]
	r.pos += n
	return out, nil
}

func (r *packetReader) readUint16() (uint16, error) {
	b, err := r.readBytes(2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

func (r *packetReader) readUint32() (uint32, error) {
	b, err := r.readBytes(4)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

func (r *packetReader) readUTF16BENetworkString() (string, error) {
	byteLen, err := r.readUint32()
	if err != nil {
		return "", err
	}

	if byteLen%2 != 0 {
		return "", fmt.Errorf("invalid utf16 byte length %d", byteLen)
	}

	if byteLen > uint32(r.remaining()) {
		return "", fmt.Errorf("string length %d exceeds remaining %d", byteLen, r.remaining())
	}

	raw, err := r.readBytes(int(byteLen))
	if err != nil {
		return "", err
	}

	codeUnits := make([]uint16, 0, len(raw)/2)

	for i := 0; i < len(raw); i += 2 {
		codeUnits = append(codeUnits, binary.BigEndian.Uint16(raw[i:i+2]))
	}

	return string(utf16.Decode(codeUnits)), nil
}

func ParsePacket(data []byte) (*Device, error) {
	if len(data) < 4+16+4+4+4+4+2 {
		return nil, errors.New("packet too small")
	}

	r := newPacketReader(data)

	magic, err := r.readBytes(4)
	if err != nil {
		return nil, err
	}

	if string(magic) != Magic {
		return nil, errors.New("invalid magic")
	}

	device := newDevice()

	token, err := r.readBytes(16)
	if err != nil {
		return nil, err
	}
	copy(device.Token[:], token)

	device.Source, err = r.readUTF16BENetworkString()
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}

	device.Action, err = r.readUTF16BENetworkString()
	if err != nil {
		return nil, fmt.Errorf("action: %w", err)
	}

	device.SoftwareName, err = r.readUTF16BENetworkString()
	if err != nil {
		return nil, fmt.Errorf("software name: %w", err)
	}

	device.SoftwareVersion, err = r.readUTF16BENetworkString()
	if err != nil {
		return nil, fmt.Errorf("software version: %w", err)
	}

	device.Port, err = r.readUint16()
	if err != nil {
		return nil, fmt.Errorf("port: %w", err)
	}

	if r.remaining() != 0 {
		return nil, fmt.Errorf("unexpected trailing bytes: %d", r.remaining())
	}

	device.RawPayload = append([]byte(nil), data...)
	device.finalize()

	return device, nil
}
