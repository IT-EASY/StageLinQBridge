package discovery

import (
	"net"
	"strconv"

	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

type Device struct {
	Source          string
	Action          string
	SoftwareName    string
	SoftwareVersion string

	IP   net.IP
	Port uint16

	Token token.Token
}

func (d Device) TokenHex() string {
	return d.Token.Hex()
}

func (d Device) Address() string {
	return net.JoinHostPort(d.IP.String(), strconv.Itoa(int(d.Port)))
}
