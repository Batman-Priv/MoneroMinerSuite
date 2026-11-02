
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const winServiceName = "MinerAgent"

const (
	createNoWindow        = 0x08000000
	createNewProcessGroup = 0x00000200
)

func xmrigProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: createNoWindow | createNewProcessGroup,
	}
}

func sendGracefulStop(p *os.Process) error {
	return p.Signal(os.Interrupt)
}

func IsElevated() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

func InstallAndStartService(execPath, configPath string, args []string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to the Service Manager (are you Administrator?): %w", err)
	}
	defer m.Disconnect()

	if s, err := m.OpenService(winServiceName); err == nil {
		_, _ = s.Control(svc.Stop)
		time.Sleep(2 * time.Second)
		_ = s.Delete()
		s.Close()
	}

	s, err := m.CreateService(winServiceName, execPath, mgr.Config{
		DisplayName: "Monero Miner Agent",
		Description: "Automatic XMRig management on this server",
		StartType:   mgr.StartAutomatic,
	}, args...)
	if err != nil {
		return err
	}
	defer s.Close()

	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
	}, 86400)

	return s.Start()
}

func UninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(winServiceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	defer s.Close()

	_, _ = s.Control(svc.Stop)
	time.Sleep(2 * time.Second)
	return s.Delete()
}

func RestartService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(winServiceName)
	if err != nil {
		return err
	}
	defer s.Close()

	_, _ = s.Control(svc.Stop)
	time.Sleep(2 * time.Second)
	return s.Start()
}

func TuneSystemForMining() {
	if err := enableLockPagesPrivilege(); err != nil {
		fmt.Printf("Note: automatic large-pages setup failed (%v). You can do it manually via gpedit.msc.\n", err)
	}
}

func enableLockPagesPrivilege() error {
	tmpDir := os.TempDir()
	cfgPath := filepath.Join(tmpDir, "miner-agent-secpol.cfg")
	dbPath := filepath.Join(tmpDir, "miner-agent-secedit.sdb")
	defer os.Remove(cfgPath)
	defer os.Remove(dbPath)

	if err := exec.Command("secedit", "/export", "/cfg", cfgPath, "/areas", "USER_RIGHTS").Run(); err != nil {
		return err
	}
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	text := string(raw)

	re := regexp.MustCompile(`(?m)^SeLockMemoryPrivilege\s*=\s*(.*)$`)
	if re.MatchString(text) {
		text = re.ReplaceAllStringFunc(text, func(line string) string {
			if strings.Contains(line, "S-1-5-18") {
				return line
			}
			return strings.TrimRight(line, "\r\n") + ",*S-1-5-18"
		})
	} else {
		text += "\r\nSeLockMemoryPrivilege = *S-1-5-18\r\n"
	}
	if err := os.WriteFile(cfgPath, []byte(text), 0644); err != nil {
		return err
	}
	return exec.Command("secedit", "/configure", "/db", dbPath, "/cfg", cfgPath, "/areas", "USER_RIGHTS").Run()
}

type winServiceHandler struct {
	stopFunc func()
}

func (h *winServiceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: accepted}
	for req := range r {
		switch req.Cmd {
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			h.stopFunc()
			changes <- svc.Status{State: svc.Stopped}
			return false, 0
		case svc.Interrogate:
			changes <- req.CurrentStatus
		}
	}
	return false, 0
}

func RunAsWindowsService(stop func()) error {
	return svc.Run(winServiceName, &winServiceHandler{stopFunc: stop})
}

func IsWindowsService() bool {
	is, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return is
}
