package mainconn

import (
	"errors"

	stageencoding "github.com/it-easy/StageLinQBridge/internal/stagelinq/encoding"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

const (
	MessageServicesAnnouncement uint32 = 0
	MessageTimeStamp            uint32 = 1
	MessageServicesRequest      uint32 = 2
)

var ErrUnknownMessage = errors.New("unknown main connection message")

type ServiceAnnouncement struct {
	Token token.Token
	Name  string
	Port  uint16
}

type TimeStamp struct {
	Token     token.Token
	PeerToken token.Token
	TimeAlive uint64
}

func BuildServicesRequest(requestToken token.Token) []byte {
	writer := stageencoding.NewWriter()

	writer.Uint32(MessageServicesRequest)
	writer.Bytes(requestToken.Bytes())

	return writer.Data()
}

func ParseMessage(data []byte) (uint32, any, error) {
	reader := stageencoding.NewReader(data)

	messageID, err := reader.Uint32()
	if err != nil {
		return 0, nil, err
	}

	tokenBytes, err := reader.Bytes(token.Size)
	if err != nil {
		return 0, nil, err
	}

	var messageToken token.Token
	copy(messageToken[:], tokenBytes)

	switch messageID {
	case MessageServicesAnnouncement:
		name, err := stageencoding.ReadNetworkStringUTF16(reader)
		if err != nil {
			return messageID, nil, err
		}

		port, err := reader.Uint16()
		if err != nil {
			return messageID, nil, err
		}

		return messageID, ServiceAnnouncement{
			Token: messageToken,
			Name:  name,
			Port:  port,
		}, nil

	case MessageTimeStamp:
		peerTokenBytes, err := reader.Bytes(token.Size)
		if err != nil {
			return messageID, nil, err
		}

		var peerToken token.Token
		copy(peerToken[:], peerTokenBytes)

		timeAlive, err := reader.Uint64()
		if err != nil {
			return messageID, nil, err
		}

		return messageID, TimeStamp{
			Token:     messageToken,
			PeerToken: peerToken,
			TimeAlive: timeAlive,
		}, nil

	case MessageServicesRequest:
		return messageID, messageToken, nil

	default:
		return messageID, nil, ErrUnknownMessage
	}
}
