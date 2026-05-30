//go:build windows

package network

import (
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ifTypeEthernetCSMACD is the Windows interface type for wired Ethernet (IEEE 802.3).
const ifTypeEthernetCSMACD = 6

// isLANInterface reports whether iface is a physical wired Ethernet adapter.
// WiFi, Bluetooth, tunnels and virtual adapters are excluded.
// This filter is only used in auto-detect mode (no lan_ip configured).
func isLANInterface(iface net.Interface) bool {
	var size uint32
	// First call: obtain required buffer size.
	_ = windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, nil, &size)
	if size == 0 {
		return false
	}

	buf := make([]byte, size)
	first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	if err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, first, &size); err != nil {
		return false
	}

	for a := first; a != nil; a = a.Next {
		if int(a.IfIndex) == iface.Index {
			return a.IfType == ifTypeEthernetCSMACD
		}
	}
	return false
}
