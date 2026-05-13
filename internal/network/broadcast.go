package network

import "net"

// BroadcastIPs returns the directed broadcast address for every active LAN
// interface. If lanIP is non-nil only the interface carrying that address is
// used; otherwise all active LAN interfaces are included. Results are
// deduplicated.
func BroadcastIPs(lanIP net.IP) ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []net.IP

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		if !isLANInterface(iface) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			if lanIP != nil && !ip.Equal(lanIP.To4()) {
				continue
			}

			mask := ipNet.Mask
			if len(mask) == net.IPv6len {
				mask = mask[12:]
			}

			broadcast := make(net.IP, net.IPv4len)
			for i := range broadcast {
				broadcast[i] = ip[i] | ^mask[i]
			}

			key := broadcast.String()
			if !seen[key] {
				seen[key] = true
				result = append(result, broadcast)
			}
		}
	}

	return result, nil
}
