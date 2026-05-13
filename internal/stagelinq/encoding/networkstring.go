package encoding

import (
	"unicode/utf16"
)

// utf16Reader is satisfied by both *Reader (slice-based) and *StreamReader
// (io.Reader-based), allowing ReadNetworkStringUTF16 to work on both.
type utf16Reader interface {
	Uint32() (uint32, error)
	Bytes(int) ([]byte, error)
}

func ReadNetworkStringUTF16(reader utf16Reader) (string, error) {
	byteLen, err := reader.Uint32() // length prefix is in bytes, not code units
	if err != nil {
		return "", err
	}

	if byteLen == 0 {
		return "", nil
	}

	data, err := reader.Bytes(int(byteLen))
	if err != nil {
		return "", err
	}

	values := make([]uint16, 0, byteLen/2)

	for i := 0; i < len(data); i += 2 {
		values = append(values, uint16(data[i])<<8|uint16(data[i+1]))
	}

	return string(utf16.Decode(values)), nil
}

func WriteNetworkStringUTF16(writer *Writer, value string) {
	encoded := utf16.Encode([]rune(value))

	writer.Uint32(uint32(len(encoded) * 2)) // length prefix is in bytes, not code units

	for _, character := range encoded {
		writer.Uint16(character)
	}
}
