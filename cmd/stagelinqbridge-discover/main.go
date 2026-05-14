package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/it-easy/StageLinQBridge/internal/bridge"
	"github.com/it-easy/StageLinQBridge/internal/config"
	"github.com/it-easy/StageLinQBridge/internal/debug"
	"github.com/it-easy/StageLinQBridge/internal/network"
	"github.com/it-easy/StageLinQBridge/internal/output"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/announce"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/beatinfo"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/discovery"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/mainconn"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/statemap"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/token"
	webui "github.com/it-easy/StageLinQBridge/web"
)

func main() {
	// --- Flags ----------------------------------------------------------

	debugFlag := flag.Bool("debug", false, "enable verbose debug logging")
	flag.Parse()

	logLevel := debug.Error // silent in normal operation — errors only
	if *debugFlag {
		logLevel = debug.Trace
	}
	logger := debug.New(logLevel)

	// --- Context --------------------------------------------------------

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	logger.Info("starting StageLinQBridge")

	// --- Config ---------------------------------------------------------

	cfgPath := resolveConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("config.json not found, using defaults", "path", cfgPath)
			cfg = config.Default()
			// Write the default config so the user can edit it.
			if saveErr := config.Save(cfgPath, cfg); saveErr != nil {
				logger.Warn("could not write default config", "error", saveErr)
			}
		} else {
			logger.Error("could not load config", "path", cfgPath, "error", err)
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

	// --- Token ----------------------------------------------------------

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
		if saveErr := config.Save(cfgPath, cfg); saveErr != nil {
			logger.Warn("could not persist token to config", "error", saveErr)
		} else {
			logger.Info("client token generated and saved", "token", clientToken.Hex())
		}
	}

	// --- Bridge (SSE hub + deck tracker) --------------------------------

	hub := bridge.NewHub()
	tracker := bridge.NewTracker(hub)

	// --- Output (sACN / Art-Net / OSC) -------------------------------------

	outMgr := output.New(cfg, logger)
	defer outMgr.Close()
	tracker.OutputFn = outMgr.Dispatch

	// --- StateMap server ------------------------------------------------

	stateMapSubscriptions := []string{
		// --- per-deck state ---
		"/Engine/Deck1/Play",
		"/Engine/Deck1/PlayState",
		"/Engine/Deck1/CurrentBPM",
		"/Engine/Deck1/Track/ArtistName",
		"/Engine/Deck1/Track/SongName",
		"/Engine/Deck1/Track/TrackName",
		"/Engine/Deck1/DeckIsMaster",
		"/Engine/Deck1/ExternalMixerVolume", // fallback for external-mixer setups
		"/Engine/Deck2/Play",
		"/Engine/Deck2/PlayState",
		"/Engine/Deck2/CurrentBPM",
		"/Engine/Deck2/Track/ArtistName",
		"/Engine/Deck2/Track/SongName",
		"/Engine/Deck2/Track/TrackName",
		"/Engine/Deck2/DeckIsMaster",
		"/Engine/Deck2/ExternalMixerVolume",
		"/Engine/Deck3/Play",
		"/Engine/Deck3/PlayState",
		"/Engine/Deck3/CurrentBPM",
		"/Engine/Deck3/Track/ArtistName",
		"/Engine/Deck3/Track/SongName",
		"/Engine/Deck3/Track/TrackName",
		"/Engine/Deck3/DeckIsMaster",
		"/Engine/Deck3/ExternalMixerVolume",
		"/Engine/Deck4/Play",
		"/Engine/Deck4/PlayState",
		"/Engine/Deck4/CurrentBPM",
		"/Engine/Deck4/Track/ArtistName",
		"/Engine/Deck4/Track/SongName",
		"/Engine/Deck4/Track/TrackName",
		"/Engine/Deck4/DeckIsMaster",
		"/Engine/Deck4/ExternalMixerVolume",
		// --- master ---
		"/Engine/Master/MasterTempo",
		// --- internal mixer (PRIME 4 standalone) ---
		"/Mixer/CH1faderPosition",
		"/Mixer/CH2faderPosition",
		"/Mixer/CH3faderPosition",
		"/Mixer/CH4faderPosition",
		"/Mixer/CrossfaderPosition",
		"/Mixer/ChannelAssignment1",
		"/Mixer/ChannelAssignment2",
		"/Mixer/ChannelAssignment3",
		"/Mixer/ChannelAssignment4",
	}

	stateMapServer, err := statemap.NewServer(logger, clientToken, stateMapSubscriptions)
	if err != nil {
		logger.Error("failed to start StateMap server", "error", err)
		os.Exit(1)
	}
	go stateMapServer.Serve(ctx)
	stateMapPort := stateMapServer.Port()
	logger.Info("StateMap server listening", "port", stateMapPort)

	go func() {
		for update := range stateMapServer.StateUpdates() {
			logger.Debug("state update", "name", update.Name, "value", update.Value)
			tracker.OnStateUpdate(update)
		}
	}()

	// --- BeatInfo server ------------------------------------------------

	beatInfoServer, err := beatinfo.NewServer(logger, clientToken)
	if err != nil {
		logger.Error("failed to start BeatInfo server", "error", err)
		os.Exit(1)
	}
	go beatInfoServer.Serve(ctx)
	beatInfoPort := beatInfoServer.Port()
	logger.Info("BeatInfo server listening", "port", beatInfoPort)

	go func() {
		for beat := range beatInfoServer.Beats() {
			tracker.OnBeatEvent(beat)
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

	// --- Announcer ------------------------------------------------------

	announcer := announce.New(logger, clientToken, lanIP, mainServer.Port())
	if err = announcer.Start(ctx); err != nil {
		logger.Error("failed to start announcer", "error", err)
		os.Exit(1)
	}

	// --- Discovery (logging only) ---------------------------------------

	listener := discovery.NewListener(logger)
	devices, err := listener.Listen(ctx)
	if err != nil {
		logger.Error("failed to start discovery listener", "error", err)
		os.Exit(1)
	}
	go func() {
		seen := make(map[string]bool)
		for device := range devices {
			if announcer.IsOwnDevice(device) || device.SoftwareName == "OfflineAnalyzer" {
				continue
			}
			key := device.TokenHex() + "@" + device.IP.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			logger.Info("device discovered",
				"source", device.Source,
				"software", device.SoftwareName,
				"version", device.SoftwareVersion,
				"ip", device.IP.String(),
			)
		}
	}()

	// --- HTTP / Web UI --------------------------------------------------

	mux := http.NewServeMux()
	mux.Handle("/events", hub)
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			RetardDiv int `json:"retardDiv"`
			TimeSig   int `json:"timeSig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.RetardDiv >= 1 {
			tracker.SetRetardDiv(body.RetardDiv)
		}
		if body.TimeSig >= 2 {
			tracker.SetTimeSig(body.TimeSig)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// POST /route — enable or disable beat output for a deck
	// Body: {"deck": 0, "active": false}
	mux.HandleFunc("/route", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Deck   int  `json:"deck"`
			Active bool `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Deck < 0 || body.Deck > 3 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		tracker.SetRoute(body.Deck, body.Active)
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /state — routing, retard config, protocol enable states (used on UI load)
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		routes := tracker.RouteActive()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"routes":    routes,
			"retardDiv": tracker.RetardDiv(),
			"timeSig":   tracker.TimeSig(),
			"protocols": outMgr.Enabled(),
		})
	})

	// POST /protocol — toggle protocol senders at runtime
	// Body: {"sacn": true, "artnet": false, "osc": true}
	mux.HandleFunc("/protocol", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		for proto, enabled := range body {
			outMgr.SetEnabled(proto, enabled)
			logger.Info("protocol toggled", "protocol", proto, "enabled", enabled)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", http.FileServer(http.FS(webui.Files)))

	httpAddr := ":8080"
	srv := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	go func() {
		logger.Info("web UI available", "url", "http://localhost"+httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Warn("web server error", "error", err)
		}
	}()
	go func() {
		time.Sleep(400 * time.Millisecond)
		launchAppWindow("http://localhost" + httpAddr)
	}()

	<-ctx.Done()
	logger.Info("StageLinQBridge stopped")
}

// resolveConfigPath returns the path to config.json.
//
// Search order:
//  1. <exe-directory>/config.json  — production layout (exe + config side-by-side)
//  2. configs/config.json          — development fallback (repo working directory)
//
// If neither exists the exe-directory path is returned so the caller can write
// a default config there.
func resolveConfigPath() string {
	// Prefer a config.json that already exists next to the running binary.
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Dev fallback: configs/config.json relative to working directory.
	if _, err := os.Stat("configs/config.json"); err == nil {
		return "configs/config.json"
	}
	// Neither exists — return exe-directory path so a default gets written there.
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "config.json")
	}
	return "config.json"
}

// launchAppWindow opens the web UI in Edge (or Chrome) app-mode — no address bar,
// no tabs, behaves like a standalone window. Falls back to the default browser.
func launchAppWindow(url string) {
	args := []string{
		"--app=" + url,
		"--window-size=1200,230",
		"--window-position=10,10",
	}

	// Look for Edge in common install paths (always present on Win10/11).
	candidates := []string{}
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if base := os.Getenv(env); base != "" {
			candidates = append(candidates,
				filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"))
		}
	}
	// Also try PATH (works if msedge is registered)
	if p, err := exec.LookPath("msedge"); err == nil {
		candidates = append([]string{p}, candidates...)
	}

	for _, exe := range candidates {
		if _, err := os.Stat(exe); err == nil {
			if err := exec.Command(exe, args...).Start(); err == nil {
				return
			}
		}
	}

	// Fallback: open in default browser (no app-mode guarantee)
	_ = exec.Command("cmd", "/C", "start", "", url).Start()
}
