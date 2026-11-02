package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		configFile = fs.String("config", "agent-config.json", "path to the agent config file (JSON)")
		listen     = fs.String("listen", "", "API address (falls back to the config file if empty)")
		token      = fs.String("token", "", "API authentication token")
		xmrigPath  = fs.String("xmrig", "", "path to the xmrig binary")
		pool       = fs.String("pool", "", "pool address(es), comma-separated for failover")
		wallet     = fs.String("wallet", "", "wallet address")
		worker     = fs.String("worker", "", "worker name")
		mode       = fs.String("mode", "", "normal | balanced | max")
		gpuFlag    = fs.String("gpu", "", "auto | cuda | opencl | none")
		tls        = fs.Bool("tls", true, "use TLS when connecting to the pool")
		save       = fs.Bool("save-config", true, "persist the final settings to disk")
		silent     = fs.Bool("silent", false, "no terminal output; log to file only (background mode)")
		autoUpdate = fs.Bool("auto-update-xmrig", true, "periodically check GitHub for a newer XMRig build and switch to it")
	)
	fs.Parse(args)

	cfg, err := LoadOrDefault(*configFile)
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	if *listen != "" {
		cfg.ListenAddr = *listen
	}
	if *token != "" {
		cfg.Token = *token
	}
	if *xmrigPath != "" {
		cfg.XMRigPath = *xmrigPath
	}
	if *pool != "" {
		cfg.PoolURLs = parsePoolList(*pool)
	}
	if *wallet != "" {
		cfg.Wallet = *wallet
	}
	if *worker != "" {
		cfg.WorkerName = *worker
	}
	if *gpuFlag != "" {
		cfg.ForceGPU = *gpuFlag
	}
	cfg.TLS = *tls

	if *mode != "" {
		m, err := ParseMode(*mode)
		if err != nil {
			log.Fatal(err)
		}
		cfg.Mode = m
	}
	if cfg.Mode == "" {
		cfg.Mode = ModeNormal  
	}

	logFile, logErr := setupLogging(cfg.WorkDir, *silent)
	if logErr == nil {
		defer logFile.Close()
	}

	if cfg.Wallet == "" || len(cfg.PoolURLs) == 0 {
		log.Fatal("wallet and pool are required — set them with --wallet and --pool or in the config file.")
	}
	if cfg.WorkerName == "" {
		hostname, _ := os.Hostname()
		cfg.WorkerName = hostname
	}
	if cfg.Token == "" {
		log.Println("warning: token is empty — this agent's API will be reachable without authentication.")
	}
	if *save {
		if err := cfg.Save(*configFile); err != nil {
			log.Printf("failed to save config: %v", err)
		}
	}

	runSupervisorAndAPI(cfg, *configFile, *autoUpdate)
}

func runSupervisorAndAPI(cfg *AgentConfig, configFile string, autoUpdate bool) {
	gpu := DetectGPU()
	log.Printf("GPU: nvidia=%v amd=%v intel=%v", gpu.HasNvidia, gpu.HasAMD, gpu.HasIntel)
	log.Printf("mode: %s | pools=%v | worker=%s", cfg.Mode, cfg.PoolURLs, cfg.WorkerName)

	sup := NewSupervisor(cfg)
	if err := sup.Start(); err != nil {
		log.Fatalf("initial miner start failed: %v", err)
	}

	startTemperatureMonitor()
	if autoUpdate {
		startXMRigAutoUpdate(sup, cfg, configFile)
	}

	api := NewAPIServer(sup, cfg)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: api.Handler()}

	go func() {
		log.Printf("agent API listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server stopped: %v", err)
		}
	}()

	stop := func() {
		log.Println("shutting down (graceful)...")
		sup.Stop()
		_ = srv.Close()
	}

	if runtime.GOOS == "windows" && IsWindowsService() {
		if err := RunAsWindowsService(stop); err != nil {
			log.Fatalf("running as a Windows service failed: %v", err)
		}
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	stop()
}

func startXMRigAutoUpdate(sup *Supervisor, cfg *AgentConfig, configFile string) {
	go func() {
		
		time.Sleep(10 * time.Minute)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			checkAndApplyXMRigUpdate(sup, cfg, configFile)
			<-ticker.C
		}
	}()
}

func checkAndApplyXMRigUpdate(sup *Supervisor, cfg *AgentConfig, configFile string) {
	rel, err := fetchLatestXMRigRelease()
	if err != nil {
		log.Printf("auto-update: failed to check the latest XMRig release: %v", err)
		return
	}
	if rel.TagName == "" || rel.TagName == cfg.XMRigVersion {
		return
	}
	log.Printf("auto-update: found a newer XMRig release (%s -> %s), downloading...", cfg.XMRigVersion, rel.TagName)

	installDir := filepath.Dir(cfg.XMRigPath)
	newPath, newVersion, err := InstallXMRig(installDir)
	if err != nil {
		log.Printf("auto-update: download failed: %v", err)
		return
	}

	cfg.XMRigPath = newPath
	cfg.XMRigVersion = newVersion
	if err := cfg.Save(configFile); err != nil {
		log.Printf("auto-update: failed to save the updated config: %v", err)
	}

	log.Printf("auto-update: restarting the miner on XMRig %s", newVersion)
	sup.Stop()
	if err := sup.Start(); err != nil {
		log.Printf("auto-update: restart failed: %v", err)
	}
}
