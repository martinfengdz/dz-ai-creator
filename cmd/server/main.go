package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dz-ai-creator/internal/pkg/core"
)

func main() {
	cfg, err := core.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	application, err := core.New(cfg)
	if err != nil {
		log.Fatalf("boot app: %v", err)
	}

	server := &http.Server{
		Addr:    listenAddr(),
		Handler: application.Router(),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sig := <-quit
	log.Printf("shutting down (signal: %v)...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("server exited gracefully")
}