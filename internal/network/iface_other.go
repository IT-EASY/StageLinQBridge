//go:build !windows

package network

import "net"

// isLANInterface accepts all broadcast-capable interfaces on non-Windows systems.
func isLANInterface(_ net.Interface) bool {
	return true
}
