package network

import (
	"fmt"
	"net"
)

// ValidateLANIP checks whether ipStr is assigned to an active LAN interface.
// Returns the parsed IPv4 address on success.
// On failure it also returns the IPs currently available on LAN interfaces so
// the caller can print a helpful error message.
func ValidateLANIP(ipStr string) (ip net.IP, available []net.IP, err error) {
	parsed := net.ParseIP(ipStr)
	if parsed == nil {
		err = fmt.Errorf("ungültige IP-Adresse: %q", ipStr)
		return
	}

	ip4 := parsed.To4()
	if ip4 == nil {
		err = fmt.Errorf("nur IPv4-Adressen werden unterstützt: %q", ipStr)
		return
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}

	// No isLANInterface filter here — when the user explicitly configures lan_ip
	// we accept any active broadcast-capable interface regardless of adapter type.
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ifaceIP := ipNet.IP.To4()
			if ifaceIP == nil {
				continue
			}

			if ifaceIP.Equal(ip4) {
				ip = ip4
				available = nil
				return
			}

			available = append(available, ifaceIP)
		}
	}

	ip = nil
	err = fmt.Errorf("IP %s ist auf keinem aktiven LAN-Interface zugewiesen", ipStr)
	return
}
