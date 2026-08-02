package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/config"
	"github.com/0xKa/equipment-checkout-system/server/db"
	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/spf13/cobra"
)

func newReconcileUsersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reconcile-users",
		Short: "Reconcile local users into Keycloak once",
		Args:  cobra.NoArgs,
		RunE:  runReconcileUsers,
	}
}

func runReconcileUsers(command *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	startupContext, cancelStartup := context.WithTimeout(
		command.Context(),
		databaseStartupTimeout,
	)
	pool, err := db.NewPool(startupContext, cfg.Database.URL, db.PoolOptions{
		MaxConnections:        cfg.Database.MaxConnections,
		MinConnections:        cfg.Database.MinConnections,
		MaxConnectionLifetime: cfg.Database.MaxConnectionLifetime,
	})
	cancelStartup()
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer pool.Close()

	queries := sqlcgen.New(pool)
	transactions := db.NewTransactionManager(pool, queries)
	identities := services.NewKeycloakIdentityAdmin(
		services.KeycloakIdentityAdminConfig{
			BaseURL:             cfg.KeycloakAdmin.BaseURL,
			Realm:               cfg.KeycloakAdmin.Realm,
			ServiceClientID:     cfg.KeycloakAdmin.ServiceClientID,
			ServiceClientSecret: cfg.KeycloakAdmin.ServiceClientSecret,
			ApplicationClientID: cfg.KeycloakAdmin.ApplicationClientID,
			Timeout:             cfg.KeycloakAdmin.Timeout,
		},
	)
	reconciler := services.NewUserReconciler(
		queries,
		transactions,
		identities,
		cfg.OIDC.IssuerURL,
	)

	reconcileContext, cancelReconcile := context.WithTimeout(
		command.Context(),
		10*time.Minute,
	)
	defer cancelReconcile()
	report, err := reconciler.Reconcile(reconcileContext)
	if err != nil {
		return err
	}

	out := command.OutOrStdout()
	fmt.Fprintf(
		out,
		"Reconciliation complete: %d updated, %d provisioned, %d failed, %d Keycloak-only orphans.\n",
		report.Updated,
		report.Provisioned,
		len(report.Failures),
		len(report.Orphans),
	)
	for _, failure := range report.Failures {
		fmt.Fprintf(out, "FAILED local user %d: %s\n", failure.UserID, failure.Reason)
	}
	for _, orphan := range report.Orphans {
		fmt.Fprintf(
			out,
			"ORPHAN Keycloak user %s (%s): manual review required\n",
			orphan.Username,
			orphan.Subject,
		)
	}
	if len(report.Failures) > 0 {
		return fmt.Errorf("user reconciliation completed with %d failures", len(report.Failures))
	}
	return nil
}
