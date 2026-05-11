package discovery

import (
	"encoding/hex"
	"net"
	"time"
)

type Device struct {
	Token           [16]byte
	TokenHex        string
	Source          string
	Action          string
	SoftwareName    string
	SoftwareVersion string
	IP              net.IP
	Port            uint16
	LastSeen        time.Time
	RawPayload      []byte
}

func newDevice() *Device {
	return &Device{}
}

func (d *Device) finalize() {
	d.TokenHex = hex.EncodeToString(d.Token[:])
}
