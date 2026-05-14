package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/it-easy/StageLinQBridge/internal/config"
	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/network"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/announce"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/beatinfo"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/discovery"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/mainconn"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/statemap"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	logger := debug.New(debug.Trace)

	logger.Info("starting StageLinQBridge discovery")

	// --- Config ---------------------------------------------------------

	cfg, err := config.Load("configs/config.json")
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("config.json nicht gefunden, verwende Standardwerte")
			cfg = config.Default()
		} else {
			logger.Error("config.json konnte nicht geladen werden", "error", err)
			os.Exit(1)
		}
	}

	// --- LAN-Interface validieren ---------------------------------------

	var lanIP net.IP

	if cfg.Network.LANIP != "" {
		var available []net.IP
		lanIP, available, err = network.ValidateLANIP(cfg.Network.LANIP)
		if err != nil {
			ips := make([]string, len(available))
			for i, ip := range available {
				ips[i] = ip.String()
			}
			hint := ""
			if len(ips) > 0 {
				hint = fmt.Sprintf(" — verfügbare LAN-IPs: %s", strings.Join(ips, ", "))
			}
			logger.Error("lan_ip ungültig", "error", fmt.Sprintf("%s%s", err, hint))
			os.Exit(1)
		}
		logger.Info("LAN-Interface konfiguriert", "lan_ip", lanIP.String())
	} else {
		logger.Warn("lan_ip nicht konfiguriert, sende auf allen aktiven LAN-Interfaces")
	}

	// --- Token & Announce -----------------------------------------------

	var clientToken token.Token
	if cfg.Network.Token != "" {
		if err := clientToken.ParseHex(cfg.Network.Token); err != nil {
			logger.Error("invalid token in config", "error", err)
			os.Exit(1)
		}
		logger.Info("client token loaded from config", "token", clientToken.Hex())
	} else {
		clientToken, err = token.NewRandom()
		if err != nil {
			logger.Error("failed to create client token", "error", err)
			os.Exit(1)
		}
		cfg.Network.Token = clientToken.Hex()
		configPath := "configs/config.json"
		if saveErr := config.Save(configPath, cfg); saveErr != nil {
			logger.Warn("could not persist token to config", "error", saveErr)
		} else {
			logger.Info("client token generated and saved", "token", clientToken.Hex())
		}
	}

	// --- StateMap server -----------------------------------------------

	stateMapSubscriptions := []string{
		"/Engine/Deck1/Play",
		"/Engine/Deck1/PlayState",
		"/Engine/Deck1/CurrentBPM",
		"/Engine/Deck1/Track/ArtistName",
		"/Engine/Deck1/Track/SongName",
		"/Engine/Deck1/Track/TrackName",
		"/Engine/Deck2/Play",
		"/Engine/Deck2/PlayState",
		"/Engine/Deck2/CurrentBPM",
		"/Engine/Deck2/Track/ArtistName",
		"/Engine/Deck2/Track/SongName",
		"/Engine/Deck2/Track/TrackName",
		"/Engine/Master/MasterTempo",
	}

	stateMapServer, err := statemap.NewServer(logger, clientToken, stateMapSubscriptions)
	if err != nil {
		logger.Error("failed to start StateMap server", "error", err)
		os.Exit(1)
	}
	go stateMapServer.Serve(ctx)
	stateMapPort := stateMapServer.Port()
	logger.Info("StateMap server listening", "port", stateMapPort)

	// Log all state updates received from the device.
	go func() {
		for update := range stateMapServer.StateUpdates() {
			logger.Info("state update",
				"ip", update.RemoteIP.String(),
				"name", update.Name,
				"value", update.Value,
			)
		}
	}()

	// --- BeatInfo server -----------------------------------------------

	beatInfoServer, err := beatinfo.NewServer(logger, clientToken)
	if err != nil {
		logger.Error("failed to start BeatInfo server", "error", err)
		os.Exit(1)
	}
	go beatInfoServer.Serve(ctx)
	beatInfoPort := beatInfoServer.Port()
	logger.Info("BeatInfo server listening", "port", beatInfoPort)

	// Log all beat events received from the device.
	go func() {
		for beat := range beatInfoServer.Beats() {
			if len(beat.Players) == 0 {
				continue
			}
			logger.Info("beat event",
				"clock", beat.Clock,
				"players", len(beat.Players),
				"deck1_beat", beat.Players[0].Beat,
				"deck1_bpm", beat.Players[0].BPM,
			)
		}
	}()

	// --- Main connection server -----------------------------------------

	mainServer, err := mainconn.NewServer(logger, clientToken, stateMapPort, beatInfoPort)
	if err != nil {
		logger.Error("failed to start main connection server", "error", err)
		os.Exit(1)
	}
	go mainServer.Serve(ctx)
	logger.Info("main connection server listening", "port", mainServer.Port())

	announcer := announce.New(logger, clientToken, lanIP, mainServer.Port())

	err = announcer.Start(ctx)
	if err != nil {
		logger.Error("failed to start announcer", "error", err)
		os.Exit(1)
	}

	// --- Discovery (logging only) --------------------------------------

	listener := discovery.NewListener(logger)

	devices, err := listener.Listen(ctx)
	if err != nil {
		logger.Error("failed to start discovery listener", "error", err)
		os.Exit(1)
	}

	go func() {
		seen := make(map[string]bool)
		for device := range devices {
			if announcer.IsOwnDevice(device) {
				continue
			}
			if device.SoftwareName == "OfflineAnalyzer" {
				continue
			}
			key := device.TokenHex() + "@" + device.IP.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			logger.Info(
				"device discovered",
				"source", device.Source,
				"software", device.SoftwareName,
				"version", device.SoftwareVersion,
				"ip", device.IP.String(),
				"port", device.Port,
				"token", device.TokenHex(),
			)
		}
	}()

	<-ctx.Done()

	logger.Info("StageLinQBridge stopped")
}
