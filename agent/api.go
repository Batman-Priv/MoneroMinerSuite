package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type apiServer struct {
	sup *Supervisor
	cfg *AgentConfig
	mux *http.ServeMux
}

func NewAPIServer(sup *Supervisor, cfg *AgentConfig) *apiServer {
	a := &apiServer{sup: sup, cfg: cfg, mux: http.NewServeMux()}
	a.mux.HandleFunc("/status", a.auth(a.handleStatus))
	a.mux.HandleFunc("/mode", a.auth(a.handleMode))
	a.mux.HandleFunc("/start", a.auth(a.handleStart))
	a.mux.HandleFunc("/stop", a.auth(a.handleStop))
	a.mux.HandleFunc("/logs", a.auth(a.handleLogs))
	a.mux.HandleFunc("/healthz", a.handleHealth) 
	return a
}

func (a *apiServer) Handler() http.Handler {
	return a.mux
}

func (a *apiServer) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.Token == "" {
			next(w, r)
			return
		}
		got := r.Header.Get("Authorization")
		want := "Bearer " + a.cfg.Token
		if got != want {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *apiServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (a *apiServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := a.sup.Status()

	if c, ok := CachedTemperature(); ok {
		status["cpu_temp_c"] = c
	}
	status["xmrig_version"] = a.cfg.XMRigVersion

	summary, err := fetchXMRigSummary(a.cfg.APIPort, a.cfg.Token)
	if err == nil {
		status["xmrig"] = summary
	} else {
		status["xmrig_error"] = err.Error()
	}
	writeJSON(w, status)
}

func (a *apiServer) handleMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	m, err := ParseMode(body.Mode)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	if err := a.sup.Reconfigure(m); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "mode changed", "mode": string(m)})
}

func (a *apiServer) handleStart(w http.ResponseWriter, r *http.Request) {
	if err := a.sup.Start(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]string{"status": "started"})
}

func (a *apiServer) handleStop(w http.ResponseWriter, r *http.Request) {
	a.sup.Stop()
	writeJSON(w, map[string]string{"status": "stopped"})
}

func (a *apiServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	n := 100
	if v := r.URL.Query().Get("lines"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			n = parsed
		}
	}
	writeJSON(w, map[string]interface{}{"logs": a.sup.Logs(n)})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fetchXMRigSummary(port int, token string) (map[string]interface{}, error) {
	client := http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/1/summary", port), nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xmrig api status %d", resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
