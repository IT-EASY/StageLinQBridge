package network

import "net"

// LANAdapter describes a physical, connected Ethernet adapter with an IPv4 address.
type LANAdapter struct {
	Name string // display name, e.g. "Ethernet 2" (Windows FriendlyName)
	IP   net.IP
}
