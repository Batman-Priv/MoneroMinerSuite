
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func ReadCPUTemperature() (float64, bool) {
	if c, ok := readTempFromSensors(); ok {
		return c, true
	}
	return readTempFromThermalZones()
}

func readTempFromSensors() (float64, bool) {
	if _, err := exec.LookPath("sensors"); err != nil {
		return 0, false
	}
	out, err := exec.Command("sensors").Output()
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "package id") && !strings.Contains(lower, "tctl") && !strings.Contains(lower, "cpu temp") {
			continue
		}
		
		idx := strings.Index(line, "+")
		if idx == -1 {
			continue
		}
		rest := line[idx+1:]
		end := strings.IndexAny(rest, "°C ")
		if end == -1 {
			continue
		}
		if v, err := strconv.ParseFloat(rest[:end], 64); err == nil {
			return v, true
		}
	}
	return 0, false
}

func readTempFromThermalZones() (float64, bool) {
	entries, err := os.ReadDir("/sys/class/thermal")
	if err != nil {
		return 0, false
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "thermal_zone") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/sys/class/thermal", e.Name(), "temp"))
		if err != nil {
			continue
		}
		milli, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			continue
		}
		return float64(milli) / 1000.0, true
	}
	return 0, false
}
