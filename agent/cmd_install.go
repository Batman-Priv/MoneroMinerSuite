package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

func cmdInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	var (
		pool    = fs.String("pool", "", "pool address(es), comma-separated for failover, e.g. primary:443,backup:443 (required)")
		wallet  = fs.String("wallet", "", "Monero wallet address (required)")
		worker  = fs.String("worker", "", "this server's name as reported to the pool (default: hostname)")
		token   = fs.String("token", "", "this agent's API token (required, use a random string)")
		mode    = fs.String("mode", "normal", "normal | balanced | max")
		gpu     = fs.String("gpu", "", "auto | cuda | opencl | none (empty = auto)")
		dir     = fs.String("dir", defaultInstallDir(), "install directory")
		listen  = fs.String("listen", "0.0.0.0:9090", "address the agent API listens on")
		tlsFlag = fs.Bool("tls", true, "use TLS when connecting to pools")
		donate  = fs.Int("donate", 1, "xmrig donate-level percentage")
		skipWalletCheck = fs.Bool("skip-wallet-check", false, "skip address format/checksum validation (not recommended)")
	)
	fs.Parse(args)

	if *pool == "" || *wallet == "" || *token == "" {
		log.Fatal("--pool, --wallet and --token are required. See: miner-agent install -h")
	}
	if !IsElevated() {
		log.Fatal("this command needs root (Linux) or Administrator (Windows) privileges.")
	}

	if *skipWalletCheck {
		log.Println("warning: wallet address validation skipped (--skip-wallet-check) — double check it's correct.")
	} else if err := ValidateMoneroAddress(*wallet); err != nil {
		log.Fatalf("wallet address failed validation: %v\n(if you're sure it's correct — e.g. a non-mainnet address — rerun with --skip-wallet-check)", err)
	} else {
		fmt.Println("==> Wallet address format and checksum look valid.")
	}

	pools := parsePoolList(*pool)
	if len(pools) == 0 {
		log.Fatal("--pool did not contain any valid entries")
	}

	workerName := *worker
	if workerName == "" {
		workerName, _ = os.Hostname()
	}
	m, err := ParseMode(*mode)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatalf("failed to create the install directory: %v", err)
	}

	fmt.Println("==> Copying the agent binary into the install directory...")
	selfPath, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to locate my own path: %v", err)
	}
	agentDest := filepath.Join(*dir, agentBinaryName())
	if err := copyExecutable(selfPath, agentDest); err != nil {
		log.Fatalf("failed to copy the agent binary: %v", err)
	}

	fmt.Println("==> Looking up and downloading the latest XMRig build for this system...")
	xmrigPath, xmrigVersion, err := InstallXMRig(*dir)
	if err != nil {
		log.Fatalf("XMRig download failed: %v", err)
	}
	fmt.Printf("    Installed: %s (%s)\n", xmrigPath, xmrigVersion)

	cfg := &AgentConfig{
		ListenAddr:   *listen,
		Token:        *token,
		XMRigPath:    xmrigPath,
		XMRigVersion: xmrigVersion,
		WorkDir:      filepath.Join(*dir, "miner-run"),
		PoolURLs:     pools,
		Wallet:       *wallet,
		WorkerName:   workerName,
		TLS:          *tlsFlag,
		DonateLevel:  *donate,
		Mode:         m,
		ForceGPU:     *gpu,
		APIPort:      16000,
	}
	cfgPath := filepath.Join(*dir, "agent-config.json")
	if err := cfg.Save(cfgPath); err != nil {
		log.Fatalf("failed to save the config: %v", err)
	}

	fmt.Println("==> Tuning the system for maximum speed (huge pages / large pages)...")
	TuneSystemForMining()

	fmt.Println("==> Registering and starting the OS service...")
	runArgs := []string{"run", "--config", cfgPath, "--silent"}
	if err := InstallAndStartService(agentDest, cfgPath, runArgs); err != nil {
		log.Fatalf("service registration failed: %v", err)
	}

	fmt.Println()
	fmt.Println("======================================================================")
	fmt.Println(" Done! This server is ready and the miner is running in the background.")
	if len(pools) > 1 {
		fmt.Printf(" %d pools configured with automatic failover.\n", len(pools))
	}
	fmt.Println(" To add it to the central panel, enter this in the web panel:")
	fmt.Println()
	fmt.Printf("   name  : %s\n", workerName)
	fmt.Printf("   addr  : %s:9090\n", firstNonLoopbackIP())
	fmt.Printf("   token : %s\n", *token)
	fmt.Printf("   os    : %s\n", runtime.GOOS)
	fmt.Println("======================================================================")
}

func firstNonLoopbackIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "<this-server-IP>"
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return "<this-server-IP>"
}
