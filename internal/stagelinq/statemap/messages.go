package statemap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	stageencoding "github.com/it-easy/StageLinQBridge/internal/stagelinq/encoding"
)

var smaaMagic = []byte{0x73, 0x6d, 0x61, 0x61} // "smaa"

var (
	subtypeEmit      = []byte{0x00, 0x00, 0x00, 0x00}
	subtypeSubscribe = []byte{0x00, 0x00, 0x07, 0xd2}
)

// StateEvent is a state value received from the device.
type StateEvent struct {
	Name  string
	Value string // raw JSON
}

// WriteSubscribe serialises a state subscribe request to w.
func WriteSubscribe(w io.Writer, name string, interval uint32) error {
	inner := stageencoding.NewWriter()
	inner.Bytes(smaaMagic)
	inner.Bytes(subtypeSubscribe)
	stageencoding.WriteNetworkStringUTF16(inner, name)
	inner.Uint32(interval)

	payload := inner.Data()

	if err := binary.Write(w, binary.BigEndian, uint32(len(payload))); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadStateEvent reads one smaa framed message. Returns nil event (no error)
// for non-emit message types (e.g. emit-response acks), so callers should skip nil.
func ReadStateEvent(r io.Reader) (*StateEvent, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	if len(payload) < 8 {
		return nil, errors.New("smaa payload too short")
	}
	if !bytes.Equal(payload[:4], smaaMagic) {
		return nil, errors.New("invalid smaa magic")
	}

	subtype := payload[4:8]
	body := stageencoding.NewStreamReader(bytes.NewReader(payload[8:]))

	switch {
	case bytes.Equal(subtype, subtypeEmit):
		name, err := stageencoding.ReadNetworkStringUTF16(body)
		if err != nil {
			return nil, err
		}
		jsonVal, err := stageencoding.ReadNetworkStringUTF16(body)
		if err != nil {
			return nil, err
		}
		return &StateEvent{Name: name, Value: jsonVal}, nil

	default:
		return nil, nil
	}
}
