package encoding

import (
	"unicode/utf16"
)

func ReadNetworkStringUTF16(reader *Reader) (string, error) {
	length, err := reader.Uint32()
	if err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	data, err := reader.Bytes(int(length) * 2)
	if err != nil {
		return "", err
	}

	values := make([]uint16, 0, length)

	for i := 0; i < len(data); i += 2 {
		value := uint16(data[i])<<8 | uint16(data[i+1])
		values = append(values, value)
	}

	return string(utf16.Decode(values)), nil
}

func WriteNetworkStringUTF16(writer *Writer, value string) {
	encoded := utf16.Encode([]rune(value))

	writer.Uint32(uint32(len(encoded)))

	for _, character := range encoded {
		writer.Uint16(character)
	}
}
