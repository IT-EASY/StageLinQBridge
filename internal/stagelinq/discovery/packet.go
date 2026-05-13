package discovery

import (
	"bytes"
	"errors"

	stageencoding "github.com/it-easy/StageLinQBridge/internal/stagelinq/encoding"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

const Magic = "airD"

var ErrInvalidMagic = errors.New("invalid discovery magic")

type Packet struct {
	Token           token.Token
	Source          string
	Action          string
	SoftwareName    string
	SoftwareVersion string
	Port            uint16
}

func ParsePacket(data []byte) (Packet, error) {
	reader := stageencoding.NewReader(data)

	magic, err := reader.Bytes(4)
	if err != nil {
		return Packet{}, err
	}

	if !bytes.Equal(magic, []byte(Magic)) {
		return Packet{}, ErrInvalidMagic
	}

	tokenBytes, err := reader.Bytes(token.Size)
	if err != nil {
		return Packet{}, err
	}

	var packetToken token.Token
	copy(packetToken[:], tokenBytes)

	source, err := stageencoding.ReadNetworkStringUTF16(reader)
	if err != nil {
		return Packet{}, err
	}

	action, err := stageencoding.ReadNetworkStringUTF16(reader)
	if err != nil {
		return Packet{}, err
	}

	softwareName, err := stageencoding.ReadNetworkStringUTF16(reader)
	if err != nil {
		return Packet{}, err
	}

	softwareVersion, err := stageencoding.ReadNetworkStringUTF16(reader)
	if err != nil {
		return Packet{}, err
	}

	port, err := reader.Uint16()
	if err != nil {
		return Packet{}, err
	}

	return Packet{
		Token:           packetToken,
		Source:          source,
		Action:          action,
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		Port:            port,
	}, nil
}

func BuildPacket(packet Packet) []byte {
	writer := stageencoding.NewWriter()

	writer.Bytes([]byte(Magic))
	writer.Bytes(packet.Token.Bytes())

	stageencoding.WriteNetworkStringUTF16(writer, packet.Source)
	stageencoding.WriteNetworkStringUTF16(writer, packet.Action)
	stageencoding.WriteNetworkStringUTF16(writer, packet.SoftwareName)
	stageencoding.WriteNetworkStringUTF16(writer, packet.SoftwareVersion)

	writer.Uint16(packet.Port)

	return writer.Data()
}
