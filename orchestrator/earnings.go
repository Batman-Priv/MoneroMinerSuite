package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type EarningsEstimate struct {
	NetworkDifficulty float64   `json:"network_difficulty"`
	PriceUSD          float64   `json:"price_usd"`
	XMRPerDay         float64   `json:"xmr_per_day"`
	USDPerDay         float64   `json:"usd_per_day"`
	UpdatedAt         time.Time `json:"updated_at"`
	Available         bool      `json:"available"`
}

var earningsState = struct {
	mu    sync.RWMutex
	value EarningsEstimate
}{}

const (
	earningsRefreshInterval = 10 * time.Minute
	blockRewardXMR          = 0.6 
)

func StartEarningsEstimator() {
	go func() {
		refreshEarningsInputs()
		ticker := time.NewTicker(earningsRefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			refreshEarningsInputs()
		}
	}()
}

func refreshEarningsInputs() {
	difficulty, diffOK := fetchNetworkDifficulty()
	price, priceOK := fetchXMRPriceUSD()

	earningsState.mu.Lock()
	earningsState.value.NetworkDifficulty = difficulty
	earningsState.value.PriceUSD = price
	earningsState.value.Available = diffOK && priceOK
	earningsState.value.UpdatedAt = time.Now()
	earningsState.mu.Unlock()
}

func EstimateForHashrate(totalHashrate float64) EarningsEstimate {
	earningsState.mu.RLock()
	e := earningsState.value
	earningsState.mu.RUnlock()

	if !e.Available || e.NetworkDifficulty <= 0 {
		return e
	}
	e.XMRPerDay = (totalHashrate * 86400 / e.NetworkDifficulty) * blockRewardXMR
	e.USDPerDay = e.XMRPerDay * e.PriceUSD
	return e
}

func fetchNetworkDifficulty() (float64, bool) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://localmonero.co/blocks/api/get_stats")
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	var out struct {
		Difficulty float64 `json:"difficulty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Difficulty <= 0 {
		return 0, false
	}
	return out.Difficulty, true
}

func fetchXMRPriceUSD() (float64, bool) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.coingecko.com/api/v3/simple/price?ids=monero&vs_currencies=usd")
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	var out struct {
		Monero struct {
			USD float64 `json:"usd"`
		} `json:"monero"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Monero.USD <= 0 {
		return 0, false
	}
	return out.Monero.USD, true
}
