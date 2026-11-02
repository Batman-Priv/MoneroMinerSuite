package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "install":
		cmdInstall(os.Args[2:])
	case "uninstall":
		cmdUninstall(os.Args[2:])
	case "config":
		cmdConfig(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`miner-agent — automatic download, run, and management of XMRig on this server

Commands:
  install    Full install: download the right XMRig build, write the config,
             and register as an OS service (systemd / Windows Service)
  config     Change settings (pool/wallet/worker/token/mode/gpu) on an existing install
  uninstall  Remove the service (files are kept unless --purge is given)
  run        Run the agent directly (this is what the installed service calls)

Quick start example:
  sudo ./miner-agent install --pool pool.supportxmr.com:443 \
      --wallet <WALLET_ADDRESS> --worker srv1 --token <RANDOM_TOKEN>

  (Windows, PowerShell as Administrator:)
  .\miner-agent.exe install --pool pool.supportxmr.com:443 --wallet <WALLET_ADDRESS> --worker srv1 --token <RANDOM_TOKEN>

For help on a specific command: miner-agent <command> -h`)
}
