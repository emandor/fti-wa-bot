package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"wa-bot-notif/internal/config"
	"wa-bot-notif/internal/httpapi"
	"wa-bot-notif/internal/storage"
	"wa-bot-notif/internal/wa"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	if lvl, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		zerolog.SetGlobalLevel(lvl)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	ctx := context.Background()

	readiness := wa.NewState(false)
	manager, err := wa.NewManager(ctx, cfg.AuthDBDSN, readiness)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize wa manager")
	}

	logStore, err := storage.NewLogStore(ctx, cfg.LogsDBDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize log store")
	}
	defer func() {
		if err := logStore.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close log store")
		}
	}()

	go logStore.StartRetention(ctx, 30*24*time.Hour)

	if err := manager.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start wa manager")
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

		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if shutdownErr := srv.Shutdown(shutCtx); shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("graceful shutdown failed")
		}
		manager.Shutdown()
		close(shutdownDone)
	}()

	log.Info().Str("addr", srv.Addr).Msg("server listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}

	<-shutdownDone
	log.Info().Msg("server stopped")
}

func addrFromPort(port int) string {
	return ":" + strconv.Itoa(port)
}
