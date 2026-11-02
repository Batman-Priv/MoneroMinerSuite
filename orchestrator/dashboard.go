package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

//go:embed web/index.html
var webFS embed.FS

type Dashboard struct {
	mu          sync.RWMutex
	servers     *ServerList
	serversPath string
	history     *HistoryStore
	auth        *PanelAuth
	sessions    *SessionStore
}

func NewDashboard(sl *ServerList, serversPath string, auth *PanelAuth) *Dashboard {
	return &Dashboard{
		servers:     sl,
		serversPath: serversPath,
		history:     NewHistoryStore(),
		auth:        auth,
		sessions:    NewSessionStore(),
	}
}

func (d *Dashboard) Serve(addr string) error {
	protected := http.NewServeMux()
	protected.HandleFunc("/", d.handleIndex)
	protected.HandleFunc("/api/servers", d.handleServers)
	protected.HandleFunc("/api/add-server", d.handleAddServer)
	protected.HandleFunc("/api/status-all", d.handleStatusAll)
	protected.HandleFunc("/api/mode", d.handleSetMode)
	protected.HandleFunc("/api/action", d.handleAction)
	protected.HandleFunc("/api/logs", d.handleLogs)
	protected.HandleFunc("/api/history", d.handleHistory)
	protected.HandleFunc("/api/earnings", d.handleEarnings)

	mux := http.NewServeMux()
	mux.HandleFunc("/login", d.handleLoginPage)
	mux.HandleFunc("/login/submit", d.handleLoginSubmit)
	mux.HandleFunc("/logout", d.handleLogout)
	mux.Handle("/", d.sessionAuthMiddleware(protected))

	d.history.StartPolling(d.servers, &d.mu)
	StartEarningsEstimator()

	if d.auth == nil {
		log.Println("warning: the panel is running WITHOUT a username/password (--allow-no-auth was set).")
	}
	log.Printf("miner management panel available at http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (d *Dashboard) sessionAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.auth == nil {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil && d.sessions.Valid(cookie.Value) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	b, err := webFS.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "index.html not found", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}

func (d *Dashboard) handleServers(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	writeJSON(w, d.servers.Servers)
}


func (d *Dashboard) handleAddServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var s Server
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	if s.Name == "" || s.Addr == "" {
		http.Error(w, `{"error":"name and addr are required"}`, http.StatusBadRequest)
		return
	}

	d.mu.Lock()
	found := false
	for i := range d.servers.Servers {
		if d.servers.Servers[i].Name == s.Name {
			d.servers.Servers[i] = s 
			found = true
			break
		}
	}
	if !found {
		d.servers.Servers = append(d.servers.Servers, s)
	}
	err := saveServers(d.serversPath, d.servers)
	d.mu.Unlock()

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "added"})
}

func (d *Dashboard) handleStatusAll(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	servers := d.servers.Servers
	d.mu.RUnlock()

	type result struct {
		Name   string                 `json:"name"`
		OS     string                 `json:"os"`
		Online bool                   `json:"online"`
		Data   map[string]interface{} `json:"data,omitempty"`
		Error  string                 `json:"error,omitempty"`
	}

	results := make([]result, len(servers))
	var wg sync.WaitGroup
	for i, s := range servers {
		wg.Add(1)
		go func(i int, s Server) {
			defer wg.Done()
			data, err := s.Status()
			if err != nil {
				results[i] = result{Name: s.Name, OS: s.OS, Online: false, Error: err.Error()}
				return
			}
			results[i] = result{Name: s.Name, OS: s.OS, Online: true, Data: data}
		}(i, s)
	}
	wg.Wait()
	writeJSON(w, results)
}

func (d *Dashboard) handleSetMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Server string `json:"server"`
		Mode   string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	d.mu.RLock()
	targets := d.pickServers(body.Server)
	d.mu.RUnlock()

	out := map[string]interface{}{}
	for _, s := range targets {
		res, err := s.SetMode(body.Mode)
		if err != nil {
			out[s.Name] = map[string]string{"error": err.Error()}
			continue
		}
		out[s.Name] = res
	}
	writeJSON(w, out)
}

func (d *Dashboard) handleAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Server string `json:"server"`
		Action string `json:"action"` 
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	d.mu.RLock()
	targets := d.pickServers(body.Server)
	d.mu.RUnlock()

	out := map[string]interface{}{}
	for _, s := range targets {
		var res map[string]interface{}
		var err error
		if body.Action == "start" {
			res, err = s.Start()
		} else {
			res, err = s.Stop()
		}
		if err != nil {
			out[s.Name] = map[string]string{"error": err.Error()}
			continue
		}
		out[s.Name] = res
	}
	writeJSON(w, out)
}

func (d *Dashboard) handleHistory(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("server")
	if name == "" {
		http.Error(w, `{"error":"server query param required"}`, http.StatusBadRequest)
		return
	}
	writeJSON(w, d.history.Get(name))
}

func (d *Dashboard) handleEarnings(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	servers := d.servers.Servers
	d.mu.RUnlock()

	total := totalLiveHashrate(servers)
	writeJSON(w, EstimateForHashrate(total))
}

func totalLiveHashrate(servers []Server) float64 {
	var mu sync.Mutex
	var total float64
	var wg sync.WaitGroup
	for _, s := range servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			data, err := s.Status()
			if err != nil {
				return
			}
			xmrig, ok := data["xmrig"].(map[string]interface{})
			if !ok {
				return
			}
			hr, ok := xmrig["hashrate"].(map[string]interface{})
			if !ok {
				return
			}
			t, ok := hr["total"].([]interface{})
			if !ok || len(t) == 0 {
				return
			}
			v, ok := t[0].(float64)
			if !ok {
				return
			}
			mu.Lock()
			total += v
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return total
}

func (d *Dashboard) handleLogs(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("server")
	d.mu.RLock()
	var target *Server
	for i := range d.servers.Servers {
		if d.servers.Servers[i].Name == name {
			target = &d.servers.Servers[i]
			break
		}
	}
	d.mu.RUnlock()
	if target == nil {
		http.Error(w, `{"error":"server not found"}`, 404)
		return
	}
	res, err := target.Logs(200)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	writeJSON(w, res)
}

func (d *Dashboard) pickServers(name string) []Server {
	if name == "" || name == "*" {
		return d.servers.Servers
	}
	for _, s := range d.servers.Servers {
		if s.Name == name {
			return []Server{s}
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
