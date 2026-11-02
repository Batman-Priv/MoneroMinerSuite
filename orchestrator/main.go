package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var (
		serversFile   = flag.String("servers", "servers.json", "server list file (starts empty if missing — add servers from the panel)")
		listen        = flag.String("listen", "0.0.0.0:8080", "web panel address")
		cliMode       = flag.String("set-mode", "", "instead of running the panel, just change the mode: normal|balanced|max")
		cliServer     = flag.String("server", "*", "target server for --set-mode or --status (* = all)")
		cliStatus     = flag.Bool("status", false, "print status and exit (without running the panel)")
		panelUser     = flag.String("panel-user", "", "panel username (if not using --panel-auth, pass it directly here)")
		panelPass     = flag.String("panel-pass", "", "panel password (for quick runs only; prefer --init-panel-auth)")
		panelAuthFile = flag.String("panel-auth", "panel-auth.json", "file storing the panel's hashed username/password")
		initPanelAuth = flag.Bool("init-panel-auth", false, "create a panel-auth.json using --panel-user and --panel-pass, then exit")
		allowNoAuth   = flag.Bool("allow-no-auth", false, "explicitly allow running the panel with NO login protection (dangerous — only for trusted, isolated networks)")
	)
	flag.Parse()

	if *initPanelAuth {
		if *panelUser == "" || *panelPass == "" {
			log.Fatal("--init-panel-auth also needs --panel-user and --panel-pass.")
		}
		auth, err := NewPanelAuth(*panelUser, *panelPass)
		if err != nil {
			log.Fatal(err)
		}
		if err := auth.Save(*panelAuthFile); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Created: %s — run without --panel-pass from now on, it will read this file.\n", *panelAuthFile)
		return
	}

	sl, err := LoadServers(*serversFile)
	if err != nil {
		log.Fatal(err)
	}

	if *cliStatus || *cliMode != "" {
		if len(sl.Servers) == 0 {
			log.Fatal("no servers defined in servers.json")
		}
	}

	if *cliStatus {
		for _, s := range sl.Servers {
			data, err := s.Status()
			if err != nil {
				fmt.Printf("[%s] offline: %v\n", s.Name, err)
				continue
			}
			fmt.Printf("[%s] %+v\n", s.Name, data)
		}
		return
	}

	if *cliMode != "" {
		targets := sl.Servers
		if *cliServer != "*" {
			targets = nil
			for _, s := range sl.Servers {
				if s.Name == *cliServer {
					targets = append(targets, s)
				}
			}
		}
		for _, s := range targets {
			res, err := s.SetMode(*cliMode)
			if err != nil {
				fmt.Printf("[%s] error: %v\n", s.Name, err)
				continue
			}
			fmt.Printf("[%s] %+v\n", s.Name, res)
		}
		return
	}

	var auth *PanelAuth
	if a, loadErr := LoadPanelAuth(*panelAuthFile); loadErr == nil {
		auth = a
		log.Printf("loaded panel login from %s (user: %s)", *panelAuthFile, auth.Username)
	} else if *panelUser != "" && *panelPass != "" {
		a, err := NewPanelAuth(*panelUser, *panelPass)
		if err != nil {
			log.Fatal(err)
		}
		auth = a
	} else if !*allowNoAuth {
		log.Fatalf(
			"refusing to start: no panel login is configured.\n\n"+
				"Couldn't read %s (%v) and no --panel-user/--panel-pass were given.\n\n"+
				"Fix one of these:\n"+
				"  - Run this once to create it:\n"+
				"    miner-orchestrator --init-panel-auth --panel-user admin --panel-pass <your-password> --panel-auth %s\n"+
				"    then rerun this exact command from the same working directory, or pass\n"+
				"    an absolute --panel-auth path so it can always be found.\n"+
				"  - Or pass --panel-user and --panel-pass directly on this command.\n"+
				"  - Or, only if you fully understand the risk, pass --allow-no-auth.\n",
			*panelAuthFile, loadErr, *panelAuthFile,
		)
	} else {
		log.Println("WARNING: starting with --allow-no-auth — the panel has NO login protection.")
	}

	dash := NewDashboard(sl, *serversFile, auth)
	if err := dash.Serve(*listen); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
