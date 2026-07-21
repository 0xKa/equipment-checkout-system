package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/config"
	"github.com/0xKa/equipment-checkout-system/server/db"
	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/handlers"
	"github.com/0xKa/equipment-checkout-system/server/logger"
	"github.com/0xKa/equipment-checkout-system/server/middleware"
	"github.com/0xKa/equipment-checkout-system/server/routes"
	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/labstack/echo/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	databaseStartupTimeout  = 5 * time.Second
	gracefulShutdownTimeout = 10 * time.Second
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API",
		Args:  cobra.NoArgs,
		RunE:  runServe,
	}
}

func runServe(_ *cobra.Command, _ []string) (runErr error) {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() {
		// Zap sinks can report os.ErrInvalid on Windows even after flushing.
		if syncErr := log.Sync(); syncErr != nil && !errors.Is(syncErr, os.ErrInvalid) {
			runErr = errors.Join(runErr, fmt.Errorf("flush logger: %w", syncErr))
		}
	}()

	databaseContext, cancelDatabaseStartup := context.WithTimeout(
		context.Background(),
		databaseStartupTimeout,
	)
	pool, err := db.NewPool(databaseContext, cfg.DatabaseURL, db.PoolOptions{
		MaxConnections:        cfg.DBMaxConnections,
		MinConnections:        cfg.DBMinConnections,
		MaxConnectionLifetime: cfg.DBMaxConnectionLifetime,
	})
	cancelDatabaseStartup()
	if err != nil {
		log.Error("database startup failed", zap.Error(err))
		return fmt.Errorf("initialize database: %w", err)
	}
	defer pool.Close()

	queries := sqlcgen.New(pool)

	itemService := services.NewItemService(queries)
	itemHandler := handlers.NewItems(itemService)

	categoryService := services.NewCategoryService(queries)
	categoryHandler := handlers.NewCategories(categoryService)

	healthHandler := handlers.NewHealth()

	server := echo.New()
	server.Logger = logger.AsSlog(log)
	server.HTTPErrorHandler = handlers.HTTPErrorHandler
	middleware.Register(server)
	routes.Register(server, healthHandler, itemHandler, categoryHandler)

	address := cfg.HTTPAddress()
	log.Info("starting HTTP server",
		zap.String("environment", cfg.AppEnv),
		zap.String("address", address),
	)

	shutdownContext, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	shutdownErrors := make(chan error, 1)
	startConfig := echo.StartConfig{
		Address:         address,
		HideBanner:      true,
		HidePort:        true,
		GracefulTimeout: gracefulShutdownTimeout,
		OnShutdownError: func(err error) {
			shutdownErrors <- err
		},
	}

	if err := startConfig.Start(shutdownContext, server); err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}

	select {
	case err := <-shutdownErrors:
		return fmt.Errorf("graceful shutdown: %w", err)
	default:
		log.Info("HTTP server stopped")
		return nil
	}
}
