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
	"sync/atomic"
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
	// Zeigt einen Setup-Dialog wenn lan_ip fehlt, falsch ist oder der Adapter
	// gerade nicht verbunden ist. Schleife damit der Dialog bei Abbruch + Neustart
	// erneut erscheinen kann.

	var lanIP net.IP

	for {
		if cfg.Network.LANIP == "" {
			logger.Warn("lan_ip nicht konfiguriert — Setup-Dialog wird geöffnet")
			runSetupDialog(cfgPath, cfg, logger)
			if cfg, err = config.Load(cfgPath); err != nil {
				logger.Error("config-Reload nach Setup fehlgeschlagen", "error", err)
				os.Exit(1)
			}
			continue
		}
		var validateErr error
		lanIP, _, validateErr = network.ValidateLANIP(cfg.Network.LANIP)
		if validateErr == nil {
			logger.Info("LAN-Interface konfiguriert", "lan_ip", lanIP.String())
			break
		}
		logger.Warn("lan_ip ungültig oder Adapter nicht verbunden — Setup-Dialog wird geöffnet",
			"ip", cfg.Network.LANIP, "error", validateErr)
		runSetupDialog(cfgPath, cfg, logger)
		if cfg, err = config.Load(cfgPath); err != nil {
			logger.Error("config-Reload nach Setup fehlgeschlagen", "error", err)
			os.Exit(1)
		}
	}

	// --- Token ----------------------------------------------------------

	// The token identifies this app to the PRIME 4.
	// Default: the known StageLinQBridge token, accepted by PRIME 4 without
	// any confirmation dialog. Persisted in config.json so it can be overridden
	// manually if needed (e.g. two instances on the same network).
	var clientToken token.Token
	if cfg.Network.Token != "" {
		if err := clientToken.ParseHex(cfg.Network.Token); err != nil {
			logger.Error("invalid token in config", "error", err)
			os.Exit(1)
		}
		logger.Info("client token loaded from config", "token", clientToken.Hex())
	} else {
		clientToken = token.StageLinQBridge
		cfg.Network.Token = clientToken.Hex()
		if saveErr := config.Save(cfgPath, cfg); saveErr != nil {
			logger.Warn("could not persist token to config", "error", saveErr)
		} else {
			logger.Info("client token set to default and saved", "token", clientToken.Hex())
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

	// --- Token-Rotation Watchdog ----------------------------------------
	// If the PRIME 4 blacklists a token (connection attempt failed before our
	// TCP Accept loop was ready), it will silently ignore all subsequent HOWDYs
	// with that token. The watchdog detects "no incoming main connection within
	// 3 s after a HOWDY" and rotates to a fresh token, giving the PRIME 4 a
	// clean slate.

	go func() {
		peerCh := mainServer.PeerConnected()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-peerCh:
				if !ok {
					return
				}
				logger.Info("PRIME 4 connected — token rotation watchdog stopped")
				return
			case <-time.After(3 * time.Second):
				newTok, err := token.NewRandom()
				if err != nil {
					logger.Warn("token rotation: could not generate new token", "error", err)
					continue
				}
				logger.Info("no PRIME 4 connection, rotating token", "new_token", newTok.Hex())
				announcer.RotateToken(newTok)
				mainServer.UpdateToken(newTok)
				stateMapServer.UpdateToken(newTok)
				cfg.Network.Token = newTok.Hex()
				if saveErr := config.Save(cfgPath, cfg); saveErr != nil {
					logger.Warn("could not save rotated token", "error", saveErr)
				}
			}
		}
	}()

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
			"mode":      tracker.Mode(),
			"beatType":  tracker.BeatType(),
		})
	})

	// POST /mode — switch between "3ch" and "1ch" output mode
	mux.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		tracker.SetMode(body.Mode)
		w.WriteHeader(http.StatusNoContent)
	})

	// POST /beattype — select active beat type for 1ch mode ("beat", "onset", "retard")
	mux.HandleFunc("/beattype", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Type string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		tracker.SetBeatType(body.Type)
		w.WriteHeader(http.StatusNoContent)
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
	// POST /shutdown — terminates the application
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		go func() {
			time.Sleep(150 * time.Millisecond) // let the response flush
			os.Exit(0)
		}()
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
//  1. <exe-directory>/config.json  — always preferred (production + dev)
//  2. configs/config.json          — only when running via "go run" (exe in TempDir)
//
// If neither exists the exe-directory path is returned so the caller can write
// a default config there.
func resolveConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	exeDir := filepath.Dir(exe)

	// Always prefer config.json next to the running binary.
	primary := filepath.Join(exeDir, "config.json")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}

	// Dev fallback ONLY for "go run": the temp binary lives inside os.TempDir().
	// A compiled binary in the repo dir must NOT fall through here — it would
	// pick up the developer's personal configs/config.json with a hardcoded IP.
	if strings.HasPrefix(strings.ToLower(exeDir), strings.ToLower(os.TempDir())) {
		if _, err := os.Stat("configs/config.json"); err == nil {
			return "configs/config.json"
		}
	}

	return primary
}

// launchWindow opens url in Edge (or Chrome) app-mode with the given dimensions.
// Falls back to the default browser if no Chromium-based browser is found.
func launchWindow(url string, width, height, x, y int) {
	args := []string{
		"--app=" + url,
		fmt.Sprintf("--window-size=%d,%d", width, height),
		fmt.Sprintf("--window-position=%d,%d", x, y),
	}

	candidates := []string{}
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if base := os.Getenv(env); base != "" {
			candidates = append(candidates,
				filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"))
		}
	}
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
	_ = exec.Command("cmd", "/C", "start", "", url).Start()
}

// launchAppWindow opens the main web UI at the standard dimensions.
func launchAppWindow(url string) { launchWindow(url, 1200, 260, 10, 10) }

// setupHTML is the self-contained HTML page for the LAN-adapter setup dialog.
const setupHTML = `<!DOCTYPE html>
<html lang="de"><head>
<meta charset="UTF-8"><title>StageLinQBridge · Setup</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#111;color:#eee;font-family:monospace;font-size:13px;
     display:flex;flex-direction:column;justify-content:center;
     height:100vh;padding:22px 24px;gap:12px;user-select:none;overflow:hidden}
h1{font-size:10px;color:#555;letter-spacing:3px;text-transform:uppercase}
p{font-size:12px;color:#999}
select{width:100%;background:#1a1a1a;border:1px solid #3a3a3a;color:#eee;
       padding:7px 10px;font-family:monospace;font-size:13px;border-radius:5px;cursor:pointer}
select:focus{outline:none;border-color:#3c6}
.btns{display:flex;gap:8px}
button{flex:1;padding:8px;font-family:monospace;font-size:13px;cursor:pointer;
       border-radius:5px;border:1px solid;transition:all .1s;font-weight:bold}
#btn-ok{background:#152015;border-color:#3c6;color:#5d5}
#btn-ok:hover{background:#1e3a1e;border-color:#5d5}
#btn-ok:disabled{opacity:.3;cursor:default}
#btn-exit{background:#1a0a0a;border-color:#6a2020;color:#d55}
#btn-exit:hover{border-color:#d55;color:#f77}
.warn{font-size:11px;color:#a60;display:none}
</style></head><body>
<h1>StageLinQBridge · Netzwerk-Setup</h1>
<p id="lbl">LAN-Adapter auswählen:</p>
<select id="sel"><option value="">Lade Adapter…</option></select>
<p class="warn" id="warn"></p>
<div class="btns">
  <button id="btn-ok" disabled>&#10003; Übernehmen</button>
  <button id="btn-exit">&#10005; Beenden</button>
</div>
<script>
fetch('/adapters').then(function(r){return r.json();}).then(function(list){
  var sel=document.getElementById('sel');
  sel.innerHTML='';
  if(!list||list.length===0){
    document.getElementById('lbl').textContent='Kein LAN-Adapter gefunden.';
    var w=document.getElementById('warn');
    w.style.display='';
    w.textContent='Ethernet-Kabel anschließen und App neu starten.';
    return;
  }
  list.forEach(function(a){
    var o=document.createElement('option');
    o.value=a.ip; o.textContent=a.name+'  —  '+a.ip;
    sel.appendChild(o);
  });
  document.getElementById('btn-ok').disabled=false;
}).catch(function(){
  document.getElementById('lbl').textContent='Fehler beim Laden der Adapter.';
});
document.getElementById('btn-ok').addEventListener('click',function(){
  var ip=document.getElementById('sel').value; if(!ip)return;
  this.disabled=true; this.textContent='Speichere…';
  fetch('/apply',{method:'POST',headers:{'Content-Type':'application/json'},
    body:JSON.stringify({ip:ip})}).then(function(){window._applied=true;window.close();});
});
document.getElementById('btn-exit').addEventListener('click',function(){
  fetch('/cancel',{method:'POST'}).finally(function(){window.close();});
});
// pagehide nur als Cancel werten wenn /apply noch nicht erfolgreich war
window.addEventListener('pagehide',function(){
  if(!window._applied) navigator.sendBeacon('/cancel');
});
</script></body></html>`

// runSetupDialog starts a temporary HTTP server, opens a compact Edge app-mode
// window listing all available LAN adapters, and blocks until the user either
// picks one (config saved) or closes/cancels (os.Exit).
func runSetupDialog(cfgPath string, cfg *config.Config, logger *debug.Logger) {
	adapters, err := network.ListLANAdapters()
	if err != nil {
		logger.Warn("LAN-Adapter konnten nicht aufgelistet werden", "error", err)
	}

	type adapterJSON struct {
		Name string `json:"name"`
		IP   string `json:"ip"`
	}
	adapterList := make([]adapterJSON, len(adapters))
	for i, a := range adapters {
		adapterList[i] = adapterJSON{Name: a.Name, IP: a.IP.String()}
	}

	done := make(chan struct{})
	var applied int32 // 1 nach erfolgreichem /apply — verhindert Doppel-Exit durch pagehide
	mux := http.NewServeMux()

	mux.HandleFunc("/adapters", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(adapterList)
	})
	mux.HandleFunc("/apply", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			IP string `json:"ip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IP == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		cfg.Network.LANIP = body.IP
		if saveErr := config.Save(cfgPath, cfg); saveErr != nil {
			logger.Error("config konnte nicht gespeichert werden", "error", saveErr)
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		logger.Info("lan_ip via Setup-Dialog gesetzt", "ip", body.IP)
		w.WriteHeader(http.StatusNoContent)
		atomic.StoreInt32(&applied, 1)
		go func() { time.Sleep(100 * time.Millisecond); close(done) }()
	})
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		if atomic.LoadInt32(&applied) == 0 { // nur beenden wenn /apply noch nicht kam
			go func() { time.Sleep(100 * time.Millisecond); os.Exit(0) }()
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, setupHTML)
	})

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		logger.Error("Setup-Server konnte nicht gestartet werden", "error", err)
		os.Exit(1)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	go func() {
		time.Sleep(400 * time.Millisecond)
		launchWindow(fmt.Sprintf("http://localhost:%d", port), 520, 210, 400, 300)
	}()

	<-done
	_ = srv.Close()
}
