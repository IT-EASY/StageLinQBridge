package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Network NetworkConfig `json:"network"`
	SACN    SACNConfig    `json:"sacn"`
	ArtNet  ArtNetConfig  `json:"artnet"`
	OSC     OSCConfig     `json:"osc"`
}

type NetworkConfig struct {
	LANIP string `json:"lan_ip"`
	Token string `json:"token,omitempty"`
}

// ChannelMap maps the three beat-event types to DMX channel numbers (1-based).
type ChannelMap struct {
	Beat      uint16 `json:"beat"`
	Downbeat  uint16 `json:"downbeat"`
	Slow      uint16 `json:"slow"` // also known as "retard" — see README
}

// SACNConfig configures E1.31 sACN output (multicast, no unicast needed).
type SACNConfig struct {
	Enabled  bool       `json:"enabled"`
	Universe uint16     `json:"universe"`  // 1–63999
	Channels ChannelMap `json:"channels"`
	PulseMS  int        `json:"pulse_ms"`  // how long the channel stays at 255
}

// ArtNetConfig configures Art-Net ArtDmx output (unicast to a single target).
// Target must be set (e.g. "192.168.1.50") — no broadcast, by design.
type ArtNetConfig struct {
	Enabled  bool       `json:"enabled"`
	Target   string     `json:"target"`   // "ip" or "ip:port" (default port 6454)
	Universe uint16     `json:"universe"` // 0–32767 (port-address)
	Channels ChannelMap `json:"channels"`
	PulseMS  int        `json:"pulse_ms"`
}

// AddressMap maps the three beat-event types to OSC address strings.
type AddressMap struct {
	Beat     string `json:"beat"`
	Downbeat string `json:"downbeat"`
	Slow     string `json:"slow"`
}

// OSCConfig configures OSC output (UDP unicast to a single target).
type OSCConfig struct {
	Enabled   bool       `json:"enabled"`
	Target    string     `json:"target"`    // "ip:port"
	Addresses AddressMap `json:"addresses"`
	PulseMS   int        `json:"pulse_ms"`
}

func Default() *Config {
	return &Config{
		SACN: SACNConfig{
			Enabled:  false,
			Universe: 1,
			Channels: ChannelMap{Beat: 1, Downbeat: 2, Slow: 11},
			PulseMS:  50,
		},
		ArtNet: ArtNetConfig{
			Enabled:  false,
			Target:   "",
			Universe: 0,
			Channels: ChannelMap{Beat: 1, Downbeat: 2, Slow: 11},
			PulseMS:  50,
		},
		OSC: OSCConfig{
			Enabled: false,
			Target:  "192.168.1.100:9000",
			Addresses: AddressMap{
				Beat:     "/stagelinq/beat",
				Downbeat: "/stagelinq/downbeat",
				Slow:     "/stagelinq/slow",
			},
			PulseMS: 50,
		},
	}
}

func Save(path string, cfg *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg := Default()
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
