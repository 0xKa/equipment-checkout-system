package cmd

import (
	"fmt"

	"github.com/0xKa/equipment-checkout-system/server/config"
	"github.com/0xKa/equipment-checkout-system/server/handlers"
	"github.com/0xKa/equipment-checkout-system/server/logger"
	"github.com/0xKa/equipment-checkout-system/server/routes"
	"github.com/labstack/echo/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API",
		Args:  cobra.NoArgs,
		RunE:  runServe,
	}
}

func runServe(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() {
		_ = log.Sync()
	}()

	healthHandler := handlers.NewHealth()
	server := echo.New()
	routes.Register(server, healthHandler)

	address := cfg.HTTPAddress()
	log.Info("starting HTTP server",
		zap.String("environment", cfg.AppEnv),
		zap.String("address", address),
	)

	if err := server.Start(address); err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}
	return nil
}
