package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"leverless-overlay/internal/config"
	"leverless-overlay/internal/input"
	"leverless-overlay/internal/ws"
)

func main() {
	configFlag := flag.String("config", "", "config file name in configs directory")
	flag.Parse()

	cwd, _ := os.Getwd()
	base := cwd
	// allow running from repo root
	if filepath.Base(cwd) != "input-cast" {
		base = filepath.Join(cwd)
	}

	configsDir := filepath.Join(base, "configs")
	activeProfilePath := filepath.Join(configsDir, ".active-profile")
	configName := *configFlag
	if configName == "" {
		configName = loadActiveProfile(activeProfilePath)
	}
	cfgPath, currentProfile := resolveConfigPath(configsDir, configName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("failed load config: %v", err)
		cfg = &config.Config{}
	}

	hub := ws.NewHub()
	// input manager and providers
	manager := input.NewManager()
	// subscribe hub broadcast to manager notifications
	manager.Subscribe(func(s *input.State) {
		msg := map[string]interface{}{"type": "input", "device_id": s.DeviceID, "timestamp": s.Timestamp, "buttons": s.Buttons}
		if bm, err := json.Marshal(msg); err == nil {
			hub.Broadcast(bm)
		}
	})
	// MVP uses only mock input. Additional providers can be registered here later
	// without changing the HTTP/WebSocket overlay path.
	mp := input.NewMockProvider("mock", manager.HandleRaw)
	manager.Register(mp)
	gp := input.NewGamepadProvider("gamepad", manager.HandleRaw)
	manager.Register(gp)
	ep := input.NewEvdevProvider("oshid", manager.HandleRaw)
	manager.Register(ep)

	// static files
	fs := http.FileServer(http.Dir(filepath.Join(base, "web", "overlay")))
	http.Handle("/static/", noStore(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(base, "web", "overlay"))))))
	http.Handle("/gamepad-static/", noStore(http.StripPrefix("/gamepad-static/", http.FileServer(http.Dir(filepath.Join(base, "web", "gamepad"))))))
	http.HandleFunc("/overlay", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(base, "web", "overlay", "index.html"))
	})
	http.HandleFunc("/preview", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(base, "web", "overlay", "index.html"))
	})
	http.HandleFunc("/gamepad", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(base, "web", "gamepad", "index.html"))
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := ws.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.AddConn(c)

		// send current config
		if cfg != nil {
			msg := map[string]interface{}{"type": "config", "config": cfg}
			_ = c.WriteJSON(msg)
		}
		// send last-state for each provider
		infos := manager.ProvidersInfo()
		for _, pi := range infos {
			if pi.Last != nil {
				im := map[string]interface{}{"type": "input", "device_id": pi.ID}
				if pi.Last.Buttons != nil {
					im["buttons"] = pi.Last.Buttons
				}
				if pi.Last.Timestamp != "" {
					im["timestamp"] = pi.Last.Timestamp
				}
				_ = c.WriteJSON(im)
			}
		}

		// read loop to detect close
		for {
			if _, _, err := c.NextReader(); err != nil {
				break
			}
		}

		hub.RemoveConn(c)
		c.Close()
	})

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			b, _ := json.Marshal(cfg)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Config-Profile", currentProfile)
			w.Write(b)
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var nc config.Config
			if err := json.Unmarshal(body, &nc); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if requestedProfile := r.URL.Query().Get("profile"); requestedProfile != "" {
				cfgPath, currentProfile = resolveConfigPath(configsDir, requestedProfile)
			}
			cfg = &nc
			_ = config.Save(cfgPath, cfg)
			_ = saveActiveProfile(activeProfilePath, currentProfile)
			// Broadcast new config to connected overlays
			msg := map[string]interface{}{"type": "config", "config": nc}
			if bm, err := json.Marshal(msg); err == nil {
				hub.Broadcast(bm)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/config/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		profiles, err := listConfigProfiles(configsDir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"current":  currentProfile,
			"profiles": profiles,
		})
	})

	http.HandleFunc("/api/config/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		nextPath, nextProfile := resolveConfigPath(configsDir, req.Name)
		nextCfg, err := config.Load(nextPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		cfgPath = nextPath
		currentProfile = nextProfile
		cfg = nextCfg
		_ = saveActiveProfile(activeProfilePath, currentProfile)
		msg := map[string]interface{}{"type": "config", "config": cfg}
		if bm, err := json.Marshal(msg); err == nil {
			hub.Broadcast(bm)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"current": currentProfile})
	})

	http.HandleFunc("/api/mapping", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			b, _ := json.Marshal(cfg.Mappings)
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var m map[string]map[string]string
			if err := json.Unmarshal(body, &m); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			cfg.Mappings = m
			_ = config.Save(cfgPath, cfg)
			// broadcast config so overlays update if necessary
			msg := map[string]interface{}{"type": "config", "config": cfg}
			if bm, err := json.Marshal(msg); err == nil {
				hub.Broadcast(bm)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// providers list and enable/disable
	http.HandleFunc("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			info := manager.ProvidersInfo()
			b, _ := json.Marshal(info)
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		case http.MethodPost:
			// body: {"id":"mock","enabled":true}
			body, _ := io.ReadAll(r.Body)
			var req struct {
				ID      string `json:"id"`
				Enabled bool   `json:"enabled"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			manager.SetEnabled(req.ID, req.Enabled)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// provider events: GET /api/providers/events?id=mock
	http.HandleFunc("/api/providers/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ev := manager.Events(id)
		b, _ := json.Marshal(ev)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	http.HandleFunc("/api/input/mock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var st input.State
		if err := json.Unmarshal(body, &st); err != nil {
			// fallback: parse generic map
			var raw map[string]interface{}
			if err2 := json.Unmarshal(body, &raw); err2 != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if did, ok := raw["device_id"].(string); ok {
				st.DeviceID = did
			}
			if bmap, ok := raw["buttons"].(map[string]interface{}); ok {
				st.Buttons = make(map[string]bool)
				for k, v := range bmap {
					if vb, ok2 := v.(bool); ok2 {
						st.Buttons[k] = vb
					}
				}
			}
		}
		if err := mp.Send(&st); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/api/input/gamepad", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var st input.State
		if err := json.Unmarshal(body, &st); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		st.DeviceID = "gamepad"
		if err := gp.Send(&st); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/api/background/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("image")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		name := filepath.Base(header.Filename)
		if name == "." || name == string(filepath.Separator) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		name = time.Now().Format("20060102-150405-") + name
		dir := filepath.Join(base, "web", "overlay", "uploads")
		if err := os.MkdirAll(dir, 0755); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		out, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer out.Close()
		if _, err := io.Copy(out, file); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"path": "uploads/" + name})
	})

	// serve root static for convenience
	http.Handle("/", noStore(fs))

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func resolveConfigPath(configsDir, name string) (string, string) {
	profile := strings.TrimSpace(name)
	if profile == "" {
		profile = "default.json"
	}
	profile = filepath.Base(profile)
	if filepath.Ext(profile) == "" {
		profile += ".json"
	}
	return filepath.Join(configsDir, profile), profile
}

func listConfigProfiles(configsDir string) ([]string, error) {
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return nil, err
	}
	profiles := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(filepath.Ext(name), ".json") {
			profiles = append(profiles, name)
		}
	}
	sort.Strings(profiles)
	return profiles, nil
}

func loadActiveProfile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func saveActiveProfile(path, profile string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(profile+"\n"), 0644)
}
