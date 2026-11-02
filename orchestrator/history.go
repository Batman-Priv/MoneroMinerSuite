package main

import (
	"sync"
	"time"
)


type HistoryPoint struct {
	Time     int64   `json:"t"` 
	Hashrate float64 `json:"h"` 
}

const (
	historyPollInterval = 30 * time.Second
	historyMaxPoints     = 120 
)


type HistoryStore struct {
	mu   sync.RWMutex
	data map[string][]HistoryPoint
}

func NewHistoryStore() *HistoryStore {
	return &HistoryStore{data: make(map[string][]HistoryPoint)}
}

func (h *HistoryStore) add(server string, hashrate float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	points := append(h.data[server], HistoryPoint{Time: time.Now().Unix(), Hashrate: hashrate})
	if len(points) > historyMaxPoints {
		points = points[len(points)-historyMaxPoints:]
	}
	h.data[server] = points
}

func (h *HistoryStore) Get(server string) []HistoryPoint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	src := h.data[server]
	out := make([]HistoryPoint, len(src))
	copy(out, src)
	return out
}

func (h *HistoryStore) StartPolling(servers *ServerList, mu *sync.RWMutex) {
	go func() {
		ticker := time.NewTicker(historyPollInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.RLock()
			list := servers.Servers
			mu.RUnlock()

			for _, s := range list {
				data, err := s.Status()
				if err != nil {
					continue
				}
				xmrig, ok := data["xmrig"].(map[string]interface{})
				if !ok {
					continue
				}
				hr, ok := xmrig["hashrate"].(map[string]interface{})
				if !ok {
					continue
				}
				total, ok := hr["total"].([]interface{})
				if !ok || len(total) == 0 {
					continue
				}
				if v, ok := total[0].(float64); ok {
					h.add(s.Name, v)
				}
			}
		}
	}()
}
