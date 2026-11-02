package main

import (
	"sync"
	"time"
)

var tempState = struct {
	mu        sync.RWMutex
	celsius   float64
	available bool
}{}

func startTemperatureMonitor() {
	go func() {
		readAndCacheTemperature()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			readAndCacheTemperature()
		}
	}()
}

func readAndCacheTemperature() {
	c, ok := ReadCPUTemperature()
	tempState.mu.Lock()
	tempState.celsius = c
	tempState.available = ok
	tempState.mu.Unlock()
}

func CachedTemperature() (float64, bool) {
	tempState.mu.RLock()
	defer tempState.mu.RUnlock()
	return tempState.celsius, tempState.available
}
