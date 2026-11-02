package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

func cmdConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	var (
		dir              = fs.String("dir", defaultInstallDir(), "install directory")
		pool             = fs.String("pool", "", "new pool address(es), comma-separated for failover")
		wallet           = fs.String("wallet", "", "new wallet address")
		worker           = fs.String("worker", "", "new worker name")
		token            = fs.String("token", "", "new API token")
		mode             = fs.String("mode", "", "normal | balanced | max")
		gpu              = fs.String("gpu", "", "auto | cuda | opencl | none")
		restart          = fs.Bool("restart", true, "restart the service after the change so it takes effect")
		skipWalletCheck  = fs.Bool("skip-wallet-check", false, "skip address format/checksum validation (not recommended)")
	)
	fs.Parse(args)

	cfgPath := filepath.Join(*dir, "agent-config.json")
	cfg, err := LoadOrDefault(cfgPath)
	if err != nil {
		log.Fatalf("failed to read config from %s: %v", cfgPath, err)
	}

	changed := false
	if *pool != "" {
		pools := parsePoolList(*pool)
		if len(pools) == 0 {
			log.Fatal("--pool did not contain any valid entries")
		}
		cfg.PoolURLs = pools
		changed = true
	}
	if *wallet != "" {
		if *skipWalletCheck {
			log.Println("warning: wallet address validation skipped (--skip-wallet-check) — double check it's correct.")
		} else if err := ValidateMoneroAddress(*wallet); err != nil {
			log.Fatalf("wallet address failed validation: %v\n(if you're sure it's correct, rerun with --skip-wallet-check)", err)
		}
		cfg.Wallet = *wallet
		changed = true
	}
	if *worker != "" {
		cfg.WorkerName = *worker
		changed = true
	}
	if *token != "" {
		cfg.Token = *token
		changed = true
	}
	if *gpu != "" {
		cfg.ForceGPU = *gpu
		changed = true
	}
	if *mode != "" {
		m, err := ParseMode(*mode)
		if err != nil {
			log.Fatal(err)
		}
		cfg.Mode = m
		changed = true
	}

	if !changed {
		fmt.Println("Nothing changed. Use --pool/--wallet/--worker/--token/--mode/--gpu.")
		return
	}

	if err := cfg.Save(cfgPath); err != nil {
		log.Fatalf("failed to save config: %v", err)
	}
	fmt.Println("Config updated.")

	if !*restart {
		fmt.Println("Restart the service yourself for this to take effect (or don't pass --restart=false).")
		return
	}
	if !IsElevated() {
		fmt.Println("Run this command as root/Administrator to auto-restart the service. The config file was updated only.")
		return
	}
	if err := RestartService(); err != nil {
		log.Printf("restarting the service failed: %v (restart it manually)", err)
		return
	}
	fmt.Println("Service restarted; changes applied.")
}
