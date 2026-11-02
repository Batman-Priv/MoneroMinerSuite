
package main

import (
	"os/exec"
	"strconv"
	"strings"
)

func ReadCPUTemperature() (float64, bool) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-CimInstance -Namespace root/wmi -ClassName MSAcpi_ThermalZoneTemperature | Select-Object -First 1 -ExpandProperty CurrentTemperature)")
	out, err := cmd.Output()
	if err != nil {
		return 0, false
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return 0, false
	}
	tenthsKelvin, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	celsius := tenthsKelvin/10.0 - 273.15
	if celsius <= 0 || celsius > 150 {
		
		return 0, false
	}
	return celsius, true
}
