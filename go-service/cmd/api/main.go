package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"wa-bot-notif/go-service/internal/config"
	"wa-bot-notif/go-service/internal/httpapi"
	"wa-bot-notif/go-service/internal/storage"
	"wa-bot-notif/go-service/internal/wa"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	readiness := wa.NewState(false)
	manager, err := wa.NewManager(ctx, cfg.AuthDBDSN, readiness)
	if err != nil {
		log.Fatalf("failed to initialize wa manager: %v", err)
	}

	logStore, err := storage.NewLogStore(ctx, cfg.LogsDBDSN)
	if err != nil {
		log.Fatalf("failed to initialize log store: %v", err)
	}
	defer func() {
		if err := logStore.Close(); err != nil {
			log.Printf("failed to close log store: %v", err)
		}
	}()

	if err := manager.Start(ctx); err != nil {
		log.Fatalf("failed to start wa manager: %v", err)
	}
	defer manager.Shutdown()

	srv := &http.Server{
		Addr: addrFromPort(cfg.Port),
		Handler: httpapi.NewHandler(
			readiness,
			manager,
			manager,
			manager,
			logStore,
			httpapi.Config{AuthToken: cfg.AuthToken, GroupJID: cfg.GroupJID},
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	shutdownDone := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
			log.Printf("graceful shutdown failed: %v", shutdownErr)
		}
		manager.Shutdown()
		close(shutdownDone)
	}()

	log.Printf("go-service listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	<-shutdownDone
	log.Print("server stopped")
}

func addrFromPort(port int) string {
	return ":" + strconv.Itoa(port)
}
