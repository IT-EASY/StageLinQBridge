package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

const Size = 16

type Token [Size]byte

// Known protocol/client tokens.
// These are fixed identifiers that PRIME 4 firmware recognises without
// requiring confirmation on the device display.
var (
	// SoundSwitch — Denon's own lighting-sync application.
	SoundSwitch = MustFromHex("52fdfc072182654f163f5f0f9a621d72")

	// SC6000_1 / SC6000_2 — tokens observed from SC6000 players.
	SC6000_1 = MustFromHex("828beb02da1f4e68a6afb0b167eaf0a2")
	SC6000_2 = MustFromHex("26d238671cd64e3f80a111826ac41120")

	// Resolume — lighting/video software.
	Resolume = MustFromHex("88fa2099ac7a4f3fbc16a995dbda2a42")

	// StageLinQBridge — the token used by this application (accepted by PRIME 4).
	StageLinQBridge = MustFromHex("3a9ab443d147af88cb1c077f8039dff9")

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

func (t *Token) ParseHex(value string) error {
	parsed, err := FromHex(value)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
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
