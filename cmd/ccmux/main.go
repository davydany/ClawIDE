package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davydany/ccmux/internal/config"
	"github.com/davydany/ccmux/internal/server"
	"github.com/davydany/ccmux/internal/store"
	"github.com/davydany/ccmux/internal/tmpl"
	"github.com/davydany/ccmux/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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

	log.Println("CCMux stopped")
}
