package encoding

import (
	"encoding/binary"
	"io"
)

// StreamReader reads big-endian binary fields from an io.Reader (e.g. a
// bufio.Reader wrapping a net.Conn). Unlike Reader it does not operate on a
// fixed byte slice, so it handles TCP streams correctly.
type StreamReader struct {
	r io.Reader
}

func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{r: r}
}

func (r *StreamReader) Bytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r.r, buf)
	return buf, err
}

func (r *StreamReader) Uint16() (uint16, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf), nil
}

func (r *StreamReader) Uint32() (uint32, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf), nil
}

func (r *StreamReader) Uint64() (uint64, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf), nil
}
