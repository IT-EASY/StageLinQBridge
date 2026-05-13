package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/it-easy/StageLinQBridge/internal/config"
	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/network"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/announce"
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

	// --- Main connection server -----------------------------------------

	mainServer, err := mainconn.NewServer(logger, clientToken, stateMapPort)
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

	// --- Discovery ------------------------------------------------------

	listener := discovery.NewListener(logger)

	devices, err := listener.Listen(ctx)
	if err != nil {
		logger.Error("failed to start discovery listener", "error", err)
		os.Exit(1)
	}

	connected := make(map[string]bool)

	// connectOutbound dials the device's main TCP port and waits for service
	// announcements. Called both from UDP discovery and from PeerConnected.
	connectOutbound := func(d discovery.Device) {
		client := mainconn.NewClient(logger, d, clientToken, stateMapPort)

		if err := client.Connect(ctx); err != nil {
			logger.Error("outbound connect failed", "error", err, "device", d.IP.String())
			return
		}

		waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
		defer waitCancel()

		if port, ok := client.WaitForService(waitCtx, "StateMap"); ok {
			logger.Info("StateMap service available", "port", port)
		} else {
			logger.Warn("StateMap not announced within 10s")
		}
	}

	// discoveredDevices maps IP → Device so PeerConnected events can look up
	// the device's announced main connection port.
	var discoveredMu sync.RWMutex
	discoveredDevices := make(map[string]discovery.Device)

	// Run discovery in background; connect outbound immediately on first sight.
	go func() {
		for device := range devices {
			if announcer.IsOwnDevice(device) {
				continue
			}
			if device.Port == 0 {
				continue
			}
			if device.SoftwareName == "OfflineAnalyzer" {
				continue
			}

			key := device.TokenHex() + "@" + device.IP.String()
			if connected[key] {
				continue
			}
			connected[key] = true

			logger.Info(
				"device discovered",
				"source", device.Source,
				"action", device.Action,
				"software", device.SoftwareName,
				"version", device.SoftwareVersion,
				"ip", device.IP.String(),
				"port", device.Port,
				"token", device.TokenHex(),
			)

			discoveredMu.Lock()
			discoveredDevices[device.IP.String()] = device
			discoveredMu.Unlock()

			// Connect outbound immediately on discovery (go-stagelinq pattern).
			go connectOutbound(device)
		}
	}()

	// Also connect outbound when the PRIME 4 connects to our main server —
	// this covers the case where it connects before we discover it via UDP.
	go func() {
		for event := range mainServer.PeerConnected() {
			ipStr := event.RemoteIP.String()

			discoveredMu.RLock()
			dev, known := discoveredDevices[ipStr]
			discoveredMu.RUnlock()

			if !known {
				logger.Warn("peer event from unknown device", "ip", ipStr)
				continue
			}

			logger.Info("peer connected — initiating outbound services exchange",
				"ip", ipStr, "port", dev.Port)

			go func(d discovery.Device) {
				time.Sleep(200 * time.Millisecond)
				connectOutbound(d)
			}(dev)
		}
	}()

	<-ctx.Done()

	logger.Info("StageLinQBridge stopped")
}
