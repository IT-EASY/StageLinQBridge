//go:build !windows

package network

// ListLANAdapters is not implemented on non-Windows platforms.
// Returns an empty list so callers compile and run without modification.
func ListLANAdapters() ([]LANAdapter, error) {
	return nil, nil
}
