
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const systemdUnitPath = "/etc/systemd/system/miner-agent.service"
const systemdServiceName = "miner-agent"

func xmrigProcAttr() *syscall.SysProcAttr {
	return nil
}

func sendGracefulStop(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

func IsElevated() bool {
	return os.Geteuid() == 0
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func InstallAndStartService(execPath, configPath string, args []string) error {
	execLine := execPath
	for _, a := range args {
		execLine += fmt.Sprintf(" %q", a)
	}

	unit := fmt.Sprintf(`[Unit]
Description=Monero Miner Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=5
AmbientCapabilities=CAP_SYS_NICE CAP_SYS_RAWIO
User=root

[Install]
WantedBy=multi-user.target
`, execLine)

	if err := os.WriteFile(systemdUnitPath, []byte(unit), 0644); err != nil {
		return err
	}
	if err := run("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := run("systemctl", "enable", systemdServiceName); err != nil {
		return err
	}
	return run("systemctl", "restart", systemdServiceName)
}

func UninstallService() error {
	_ = run("systemctl", "stop", systemdServiceName)
	_ = run("systemctl", "disable", systemdServiceName)
	_ = os.Remove(systemdUnitPath)
	return run("systemctl", "daemon-reload")
}

func RestartService() error {
	return run("systemctl", "restart", systemdServiceName)
}

func TuneSystemForMining() {
	_ = run("sysctl", "-w", "vm.nr_hugepages=1280")
	_ = run("modprobe", "msr")
}

func IsWindowsService() bool { return false }

func RunAsWindowsService(stop func()) error {
	return fmt.Errorf("this binary is running on Linux; there's no Windows service here")
}
