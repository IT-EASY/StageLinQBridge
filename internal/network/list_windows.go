//go:build windows

package network

import (
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ListLANAdapters returns all physical Ethernet adapters that currently have an
// IPv4 address and are operationally up (cable connected, not disabled).
//
// Filters applied via GetAdaptersAddresses:
//   - IfType == 6           (Ethernet CSMACD — excludes WiFi, cellular, PPP)
//   - OperStatus == 1       (IfOperStatusUp — excludes disconnected / disabled)
//   - PhysicalAddressLength >= 6  (real MAC — excludes loopback, most VPN TAP adapters)
func ListLANAdapters() ([]LANAdapter, error) {
	var size uint32
	_ = windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, nil, &size)
	if size == 0 {
		return nil, nil
	}

	buf := make([]byte, size)
	first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	if err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, first, &size); err != nil {
		return nil, err
	}

	// Build index → friendly-name map for adapters that pass all filters.
	type entry struct{ name string }
	byIdx := make(map[int]entry)
	for a := first; a != nil; a = a.Next {
		if a.IfType != ifTypeEthernetCSMACD {
			continue
		}
		if a.OperStatus != 1 { // IfOperStatusUp
			continue
		}
		if a.PhysicalAddressLength < 6 {
			continue
		}
		byIdx[int(a.IfIndex)] = entry{name: windows.UTF16PtrToString(a.FriendlyName)}
	}

	// Match with IPv4 addresses from the Go net package.
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []LANAdapter
	for _, iface := range ifaces {
		e, ok := byIdx[iface.Index]
		if !ok {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			result = append(result, LANAdapter{Name: e.name, IP: ip4})
		}
	}
	return result, nil
}
