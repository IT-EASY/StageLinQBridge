package encoding

import (
	"encoding/binary"
)

type Writer struct {
	data []byte
}

func NewWriter() *Writer {
	return &Writer{
		data: make([]byte, 0),
	}
}

func (w *Writer) Bytes(value []byte) {
	w.data = append(w.data, value...)
}

func (w *Writer) Uint16(value uint16) {
	var buffer [2]byte
	binary.BigEndian.PutUint16(buffer[:], value)
	w.Bytes(buffer[:])
}

func (w *Writer) Uint32(value uint32) {
	var buffer [4]byte
	binary.BigEndian.PutUint32(buffer[:], value)
	w.Bytes(buffer[:])
}

func (w *Writer) Uint64(value uint64) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], value)
	w.Bytes(buffer[:])
}

func (w *Writer) Len() int {
	return len(w.data)
}

func (w *Writer) Data() []byte {
	result := make([]byte, len(w.data))
	copy(result, w.data)

	return result
}
