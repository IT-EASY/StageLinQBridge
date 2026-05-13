package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

const Size = 16

type Token [Size]byte

// Known protocol/client tokens.
var (
	// SoundSwitch token observed in official/client traffic.
	SoundSwitch = MustFromHex("52fdfc072182654f163f5f0f9a621d72")

	// Zero token used in specific protocol messages.
	Zero Token
)

func NewRandom() (Token, error) {
	var token Token

	_, err := rand.Read(token[:])
	if err != nil {
		return Token{}, err
	}

	// Protocol requirement:
	// MSB of first byte must not be set.
	token[0] &= 0x7F

	return token, nil
}

func FromHex(value string) (Token, error) {
	var token Token

	decoded, err := hex.DecodeString(value)
	if err != nil {
		return Token{}, err
	}

	if len(decoded) != Size {
		return Token{}, errors.New("invalid token length")
	}

	copy(token[:], decoded)

	return token, nil
}

func MustFromHex(value string) Token {
	token, err := FromHex(value)
	if err != nil {
		panic(err)
	}

	return token
}

func (t Token) Bytes() []byte {
	result := make([]byte, Size)
	copy(result, t[:])

	return result
}

func (t Token) Hex() string {
	return hex.EncodeToString(t[:])
}

func (t Token) IsZero() bool {
	for _, value := range t {
		if value != 0 {
			return false
		}
	}

	return true
}
