package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type xmrigConfig struct {
	Autosave   bool                   `json:"autosave"`
	Background bool                   `json:"background"`
	Donate     int                    `json:"donate-level"`
	LogFile    string                 `json:"log-file,omitempty"`
	CPU        xmrigCPU               `json:"cpu"`
	OpenCL     xmrigOpenCL            `json:"opencl"`
	CUDA       xmrigCUDA              `json:"cuda"`
	Pools      []xmrigPool            `json:"pools"`
	API        xmrigAPI               `json:"api"`
	HTTP       map[string]interface{} `json:"http"`
}

type xmrigCPU struct {
	Enabled        bool   `json:"enabled"`
	HugePages      bool   `json:"huge-pages"`
	HugePages1G    bool   `json:"huge-pages-jit"`
	Priority       int    `json:"priority"` 
	MaxThreadsHint int    `json:"max-threads-hint"`
	Asm            string `json:"asm"` 
	Yield          bool   `json:"yield"`
}

type xmrigOpenCL struct {
	Enabled bool `json:"enabled"`
}

type xmrigCUDA struct {
	Enabled bool `json:"enabled"`
}

type xmrigPool struct {
	URL       string `json:"url"`
	User      string `json:"user"`
	Pass      string `json:"pass"`
	KeepAlive bool   `json:"keepalive"`
	TLS       bool   `json:"tls"`
	Nicehash  bool   `json:"nicehash"`
}

type xmrigAPI struct {
	ID       string `json:"id"`
	WorkerID string `json:"worker-id"`
}

func BuildXMRigConfig(cfg *AgentConfig, gpu GPUInfo) xmrigConfig {
	pools := make([]xmrigPool, 0, len(cfg.PoolURLs))
	for _, url := range cfg.PoolURLs {
		if url == "" {
			continue
		}
		pools = append(pools, xmrigPool{
			URL:       url,
			User:      cfg.Wallet + "." + cfg.WorkerName,
			Pass:      "x",
			KeepAlive: true,
			TLS:       cfg.TLS,
		})
	}

	c := xmrigConfig{
		Autosave:   true,
		Background: false,
		Donate:     cfg.DonateLevel,
		Pools:      pools,
		API: xmrigAPI{
			ID:       cfg.WorkerName,
			WorkerID: cfg.WorkerName,
		},
		HTTP: map[string]interface{}{
			"enabled":      true,
			"host":         "127.0.0.1",
			"port":         cfg.APIPort,
			"access-token": cfg.Token,
			"restricted":   false,
		},
	}

	c.CPU = xmrigCPU{
		Enabled: true,
		Asm:     "auto",
	}

	switch cfg.Mode {
	case ModeNormal:
		c.CPU.HugePages = false
		c.CPU.Priority = 1
		c.CPU.MaxThreadsHint = 50
		c.CPU.Yield = true
	case ModeBalanced:
		c.CPU.HugePages = true
		c.CPU.Priority = 3
		c.CPU.MaxThreadsHint = 85
		c.CPU.Yield = true
	case ModeMax:
		c.CPU.HugePages = true
		c.CPU.HugePages1G = true
		c.CPU.Priority = 5
		c.CPU.MaxThreadsHint = 100
		c.CPU.Yield = false 
	}

	useGPU := cfg.ForceGPU
	if useGPU == "" {
		useGPU = "auto"
	}
	if useGPU == "auto" || useGPU == "cuda" {
		if gpu.HasNvidia {
			c.CUDA.Enabled = true
		}
	}
	if useGPU == "auto" || useGPU == "opencl" {
		
		if gpu.HasAMD || gpu.HasIntel {
			c.OpenCL.Enabled = true
		}
	}
	if useGPU == "none" {
		c.CUDA.Enabled = false
		c.OpenCL.Enabled = false
	}

	return c
}

func WriteXMRigConfig(cfg *AgentConfig, xc xmrigConfig) (string, error) {
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(cfg.WorkDir, "xmrig-config.json")
	b, err := json.MarshalIndent(xc, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return "", err
	}
	return path, nil
}

const (
	
	crashLoopThreshold = 6
	crashLoopWindow    = 3 * time.Minute
	
	quickFailThreshold = 10 * time.Second
	
	stableRunResetAfter = 2 * time.Minute
	
	gracefulStopTimeout = 15 * time.Second
)

type Supervisor struct {
	mu         sync.Mutex
	cfg        *AgentConfig
	configPath string
	cmd        *exec.Cmd
	running    bool
	gen        int 
	restarts   int
	startedAt  time.Time
	logBuf     *ringBuffer
	waitDone   chan struct{} 
	crashTimes []time.Time   
	crashLoop  bool          
}

func NewSupervisor(cfg *AgentConfig) *Supervisor {
	return &Supervisor{
		cfg:    cfg,
		logBuf: newRingBuffer(500),
	}
}

func (s *Supervisor) Reconfigure(mode Mode) error {
	s.mu.Lock()
	s.cfg.Mode = mode
	s.mu.Unlock()
	return s.restartWithNewConfig()
}

func (s *Supervisor) restartWithNewConfig() error {
	gpu := DetectGPU()
	xc := BuildXMRigConfig(s.cfg, gpu)
	path, err := WriteXMRigConfig(s.cfg, xc)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.configPath = path
	s.mu.Unlock()
	s.Stop()
	return s.Start()
}

func (s *Supervisor) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("already running")
	}
	if s.configPath == "" {
		gpu := DetectGPU()
		xc := BuildXMRigConfig(s.cfg, gpu)
		p, err := WriteXMRigConfig(s.cfg, xc)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		s.configPath = p
	}
	s.crashLoop = false
	s.crashTimes = nil
	s.gen++
	myGen := s.gen
	s.running = true
	cfgPath := s.configPath
	s.mu.Unlock()

	go s.superviseLoop(cfgPath, myGen)
	return nil
}

func (s *Supervisor) superviseLoop(cfgPath string, myGen int) {
	backoff := time.Second
	for {
		s.mu.Lock()
		if s.gen != myGen {
			s.mu.Unlock()
			return
		}
		runStart := time.Now()
		s.startedAt = runStart
		cmd := exec.Command(s.cfg.XMRigPath, "--config", cfgPath)
		cmd.SysProcAttr = xmrigProcAttr()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		done := make(chan struct{})
		s.cmd = cmd
		s.waitDone = done
		s.mu.Unlock()

		if err := cmd.Start(); err != nil {
			close(done)
			log.Printf("failed to start xmrig: %v", err)
			s.logBuf.WriteLine(fmt.Sprintf("[agent] start error: %v", err))
			if s.recordRunOutcome(runStart) {
				s.announceCrashLoop()
				return
			}
			time.Sleep(backoff)
			backoff = minDur(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second

		go s.pipeLogs(stdout)
		go s.pipeLogs(stderr)

		err := cmd.Wait()
		close(done)

		s.mu.Lock()
		if s.gen != myGen {
			
			s.mu.Unlock()
			return
		}
		s.restarts++
		s.mu.Unlock()

		if s.recordRunOutcome(runStart) {
			s.announceCrashLoop()
			return
		}

		log.Printf("xmrig exited (%v), auto-restarting...", err)
		s.logBuf.WriteLine(fmt.Sprintf("[agent] xmrig exited: %v — restarting", err))
		time.Sleep(2 * time.Second)
	}
}

func (s *Supervisor) recordRunOutcome(runStart time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	runDuration := time.Since(runStart)
	if runDuration >= stableRunResetAfter {
		s.crashTimes = nil
		return false
	}
	if runDuration >= quickFailThreshold {
		return false
	}

	now := time.Now()
	s.crashTimes = append(s.crashTimes, now)
	cutoff := now.Add(-crashLoopWindow)
	pruned := s.crashTimes[:0]
	for _, t := range s.crashTimes {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	s.crashTimes = pruned

	if len(s.crashTimes) >= crashLoopThreshold {
		s.crashLoop = true
		s.running = false
		return true
	}
	return false
}

func (s *Supervisor) announceCrashLoop() {
	msg := "auto-retry stopped: xmrig keeps crashing quickly and repeatedly " +
		"(likely a configuration error, a wrong binary path, or the pool being unreachable). " +
		"Check the config, then retry via the /start API or the config command once it's fixed."
	log.Println(msg)
	s.logBuf.WriteLine("[agent] " + msg)
}

func (s *Supervisor) pipeLogs(r io.Reader) {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		s.logBuf.WriteLine(sc.Text())
	}
}

func (s *Supervisor) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.gen++ 
	cmd := s.cmd
	done := s.waitDone
	s.running = false
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return
	}

	if err := sendGracefulStop(cmd.Process); err != nil {
		
		_ = cmd.Process.Kill()
		return
	}

	if done == nil {
		_ = cmd.Process.Kill()
		return
	}
	select {
	case <-done:
		
	case <-time.After(gracefulStopTimeout):
		log.Println("xmrig did not shut down gracefully in time; force-killing...")
		_ = cmd.Process.Kill()
	}
}

func (s *Supervisor) Status() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]interface{}{
		"running":    s.running,
		"mode":       s.cfg.Mode,
		"restarts":   s.restarts,
		"started_at": s.startedAt,
		"config":     s.configPath,
		"crash_loop": s.crashLoop,
	}
}

func (s *Supervisor) Logs(n int) []string {
	return s.logBuf.Last(n)
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

type ringBuffer struct {
	mu    sync.Mutex
	lines []string
	size  int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{lines: make([]string, 0, size), size: size}
}

func (r *ringBuffer) WriteLine(l string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, l)
	if len(r.lines) > r.size {
		r.lines = r.lines[len(r.lines)-r.size:]
	}
}

func (r *ringBuffer) Last(n int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || n > len(r.lines) {
		n = len(r.lines)
	}
	out := make([]string, n)
	copy(out, r.lines[len(r.lines)-n:])
	return out
}
