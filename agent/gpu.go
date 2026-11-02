package main

import (
	"os/exec"
	"strings"
)

type GPUInfo struct {
	HasNvidia bool 
	HasAMD    bool 
	HasIntel  bool 
}

func DetectGPU() GPUInfo {
	info := GPUInfo{}

	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		if out, err := exec.Command("nvidia-smi", "-L").Output(); err == nil && len(out) > 0 {
			info.HasNvidia = true
		}
	}

	if _, err := exec.LookPath("rocm-smi"); err == nil {
		if out, err := exec.Command("rocm-smi", "--showid").Output(); err == nil && len(out) > 0 {
			info.HasAMD = true
		}
	}

	if _, err := exec.LookPath("clinfo"); err == nil {
		if out, err := exec.Command("clinfo").Output(); err == nil {
			vendors := parseOpenCLVendors(string(out))
			if vendors["amd"] {
				info.HasAMD = true
			}
			if vendors["intel"] {
				info.HasIntel = true
			}
			
		}
	}

	return info
}

func parseOpenCLVendors(clinfoOutput string) map[string]bool {
	found := map[string]bool{}
	for _, line := range strings.Split(clinfoOutput, "\n") {
		if !strings.Contains(line, "Platform Vendor") {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "advanced micro devices"), strings.Contains(lower, "amd"):
			found["amd"] = true
		case strings.Contains(lower, "intel"):
			found["intel"] = true
		case strings.Contains(lower, "nvidia"):
			found["nvidia"] = true
		}
	}
	return found
}
