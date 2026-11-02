package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var logState = struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	silent bool
}{}

func setupLogging(workDir string, silent bool) (*os.File, error) {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(workDir, "agent.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	applyLogOutput(f, silent)

	logState.mu.Lock()
	logState.path = path
	logState.file = f
	logState.silent = silent
	logState.mu.Unlock()

	startLogRotation()
	return f, nil
}

func applyLogOutput(f *os.File, silent bool) {
	if silent {
		log.SetOutput(f)
	} else {
		log.SetOutput(io.MultiWriter(os.Stdout, f))
	}
	log.SetFlags(log.LstdFlags)
}

const (
	logRotateMaxBytes = 20 * 1024 * 1024 
	logRotateInterval = 15 * time.Minute
)

func startLogRotation() {
	go func() {
		ticker := time.NewTicker(logRotateInterval)
		defer ticker.Stop()
		for range ticker.C {
			rotateAgentLogIfLarge()
		}
	}()
}

func rotateAgentLogIfLarge() {
	logState.mu.Lock()
	defer logState.mu.Unlock()

	if logState.path == "" || logState.file == nil {
		return
	}
	info, err := logState.file.Stat()
	if err != nil || info.Size() < logRotateMaxBytes {
		return
	}

	_ = logState.file.Close()
	oldPath := logState.path + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(logState.path, oldPath); err != nil {
		log.Printf("log rotation: failed to rename %s: %v", logState.path, err)
	}

	f, err := os.OpenFile(logState.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("log rotation: failed to reopen %s: %v", logState.path, err)
		return
	}
	logState.file = f
	applyLogOutput(f, logState.silent)
}
