package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func cmdUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	var (
		purge = fs.Bool("purge", false, "also delete the entire install directory (config, logs, xmrig)")
		dir   = fs.String("dir", defaultInstallDir(), "install directory")
	)
	fs.Parse(args)

	if !IsElevated() {
		log.Fatal("this command needs root (Linux) or Administrator (Windows) privileges.")
	}

	if err := UninstallService(); err != nil {
		log.Printf("removing the service failed (maybe it wasn't installed): %v", err)
	} else {
		fmt.Println("Service removed.")
	}

	if *purge {
		if err := os.RemoveAll(*dir); err != nil {
			log.Printf("failed to remove %s: %v", *dir, err)
		} else {
			fmt.Printf("Removed %s.\n", *dir)
		}
	}
}
