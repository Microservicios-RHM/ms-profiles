package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Microservicios-RHM/ms-profiles/internal/application"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/config"
	httpapi "github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/http"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/logging"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/messaging"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/persistence"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger := logging.NewLogger(settings.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("Connecting to PostgreSQL", "host", settings.DBHost, "database", settings.DBName)
	pool, err := persistence.NewPool(
		ctx,
		settings.DatabaseDSN(),
		settings.DBConnectMaxAttempts,
		time.Duration(settings.DBConnectRetryDelayMs)*time.Millisecond,
		settings.DBPoolMax,
		logger,
	)
	if err != nil {
		logger.Error("perfiles-service failed to start", "err", err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	applied, err := persistence.RunMigrations(ctx, pool)
	if err != nil {
		logger.Error("perfiles-service failed to start", "err", err.Error())
		os.Exit(1)
	}
	for _, m := range applied {
		logger.Info("Database migration applied", "migration", m)
	}
	logger.Info("PostgreSQL connection ready")

	repository := persistence.NewProfileRepository(pool)
	createDefaultProfile := application.NewCreateDefaultProfile(repository, logger)
	syncProfile := application.NewSyncProfile(repository, logger)
	archiveProfile := application.NewArchiveProfile(repository, logger)
	getProfile := application.NewGetProfile(repository)
	updateProfile := application.NewUpdateProfile(repository)
	listProfiles := application.NewListProfiles(repository)

	consumer := messaging.NewConsumer(
		settings.BrokerURL,
		settings.BrokerExchange,
		settings.BrokerQueue,
		logger,
		settings.BrokerConnectMaxAttempts,
		time.Duration(settings.BrokerConnectRetryDelayMs)*time.Millisecond,
	)
	consumer.On("empleado.creado", createDefaultProfile.Handle)
	consumer.On("empleado.actualizado", syncProfile.Handle)
	consumer.On("empleado.retirado", archiveProfile.Handle)

	logger.Info("Connecting to RabbitMQ", "exchange", settings.BrokerExchange, "queue", settings.BrokerQueue)
	if err := consumer.Start(ctx); err != nil {
		logger.Error("perfiles-service failed to start", "err", err.Error())
		os.Exit(1)
	}
	defer consumer.Close()

	mux := httpapi.NewRouter(getProfile, updateProfile, listProfiles)
	handler := httpapi.RequestLogger(logger)(mux)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.Port),
		Handler: handler,
	}

	go func() {
		logger.Info("Profiles service started", "port", settings.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "err", err.Error())
		}
	}()

	<-ctx.Done()
	logger.Info("Graceful shutdown started")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	logger.Info("Profiles service stopped")
}
