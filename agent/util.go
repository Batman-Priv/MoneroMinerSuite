package main

import (
	"io"
	"os"
	"runtime"
	"strings"
)

func agentBinaryName() string {
	if runtime.GOOS == "windows" {
		return "miner-agent.exe"
	}
	return "miner-agent"
}

func defaultInstallDir() string {
	if runtime.GOOS == "windows" {
		return `C:\MinerAgent`
	}
	return "/opt/miner-agent"
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func parsePoolList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
