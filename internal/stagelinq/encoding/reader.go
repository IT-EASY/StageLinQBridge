package encoding

import (
	"encoding/binary"
	"errors"
)

var ErrUnexpectedEOF = errors.New("unexpected end of buffer")

type Reader struct {
	data   []byte
	offset int
}

func NewReader(data []byte) *Reader {
	return &Reader{
		data: data,
	}
}

func (r *Reader) Remaining() int {
	return len(r.data) - r.offset
}

func (r *Reader) EOF() bool {
	return r.offset >= len(r.data)
}

func (r *Reader) Offset() int {
	return r.offset
}

func (r *Reader) Seek(offset int) error {
	if offset < 0 || offset > len(r.data) {
		return ErrUnexpectedEOF
	}

	r.offset = offset
	return nil
}

func (r *Reader) Skip(count int) error {
	if count < 0 || r.Remaining() < count {
		return ErrUnexpectedEOF
	}

	r.offset += count
	return nil
}

func (r *Reader) Bytes(count int) ([]byte, error) {
	if count < 0 || r.Remaining() < count {
		return nil, ErrUnexpectedEOF
	}

	value := r.data[r.offset : r.offset+count]
	r.offset += count

	return value, nil
}

func (r *Reader) Uint16() (uint16, error) {
	value, err := r.Bytes(2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(value), nil
}

func (r *Reader) Uint32() (uint32, error) {
	value, err := r.Bytes(4)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(value), nil
}

func (r *Reader) Uint64() (uint64, error) {
	value, err := r.Bytes(8)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(value), nil
}
