package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Network NetworkConfig `json:"network"`
	SACN    SACNConfig    `json:"sacn"`
	OSC     OSCConfig     `json:"osc"`
}

type NetworkConfig struct {
	LANIP  string `json:"lan_ip"`
	Token  string `json:"token,omitempty"` // persistent StageLinQ token (hex); generated on first run
}

type SACNConfig struct {
	Mode     string `json:"mode"`               // "multicast" or "unicast"
	Universe uint16 `json:"universe"`
	TargetIP string `json:"target_ip,omitempty"` // unicast only
}

type OSCConfig struct {
	TargetIP string `json:"target_ip"`
	Port     uint16 `json:"port"`
}

func Default() *Config {
	return &Config{
		SACN: SACNConfig{
			Mode:     "multicast",
			Universe: 1,
		},
		OSC: OSCConfig{
			Port: 9000,
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
