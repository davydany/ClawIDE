package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/pidfile"
	"github.com/davydany/ClawIDE/internal/server"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmpl"
	"github.com/davydany/ClawIDE/internal/version"
	"github.com/davydany/ClawIDE/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ShowVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	log.Printf("Starting %s", version.String())

	// Single-instance enforcement via PID file
	pidPath := cfg.PidFilePath()
	existingPID, err := pidfile.Read(pidPath)
	if err == nil && pidfile.IsRunning(existingPID) {
		if cfg.Restart {
			log.Printf("Killing existing ClawIDE instance (PID %d)...", existingPID)
			if err := pidfile.Kill(existingPID); err != nil {
				log.Fatalf("Failed to kill existing instance: %v", err)
			}
			log.Println("Existing instance stopped")
		} else {
			fmt.Fprintf(os.Stderr, "\033[31mError: ClawIDE is already running (PID %d).\nUse --restart to kill the existing instance and start a new one.\033[0m\n", existingPID)
			os.Exit(1)
		}
	}

	if err := pidfile.Write(pidPath); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer pidfile.Remove(pidPath)

	st, err := store.New(cfg.StateFilePath())
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	renderer, err := tmpl.New(web.EmbeddedFS)
	if err != nil {
		log.Fatalf("Failed to initialize template renderer: %v", err)
	}

	srv := server.New(cfg, st, renderer)

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("ClawIDE stopped")
}
