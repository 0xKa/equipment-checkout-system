package services

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
)

type UserReconciliationFailure struct {
	UserID int64
	Reason string
}

type UserReconciliationReport struct {
	Updated     int
	Provisioned int
	Orphans     []types.ManagedIdentity
	Failures    []UserReconciliationFailure
}

type UserReconciler interface {
	Reconcile(ctx context.Context) (UserReconciliationReport, error)
}

type userReconciler struct {
	queries      sqlcgen.Querier
	transactions UserTransactionRunner
	identities   IdentityAdmin
	issuer       string
}

var _ UserReconciler = (*userReconciler)(nil)

func NewUserReconciler(
	queries sqlcgen.Querier,
	transactions UserTransactionRunner,
	identities IdentityAdmin,
	issuer string,
) UserReconciler {
	return &userReconciler{
		queries:      queries,
		transactions: transactions,
		identities:   identities,
		issuer:       issuer,
	}
}

func (r *userReconciler) Reconcile(
	ctx context.Context,
) (UserReconciliationReport, error) {
	report := UserReconciliationReport{
		Orphans:  make([]types.ManagedIdentity, 0),
		Failures: make([]UserReconciliationFailure, 0),
	}

	users, err := r.queries.ListUsers(ctx, nil)
	if err != nil {
		return report, utils.UnexpectedDatabaseError(ctx, "list users for reconciliation", err)
	}

	for _, user := range users {
		action, reconcileErr := r.reconcileUser(ctx, user.ID)
		if reconcileErr != nil {
			report.Failures = append(report.Failures, UserReconciliationFailure{
				UserID: user.ID,
				Reason: reconciliationFailureReason(reconcileErr),
			})
			continue
		}
		switch action {
		case "provisioned":
			report.Provisioned++
		case "updated":
			report.Updated++
		}
	}

	identities, err := r.identities.ListIdentities(ctx)
	if err != nil {
		return report, fmt.Errorf("list Keycloak users for reconciliation: %w", err)
	}
	linkedSubjects, err := r.linkedSubjects(ctx)
	if err != nil {
		return report, err
	}
	for _, identity := range identities {
		if _, linked := linkedSubjects[identity.Subject]; !linked {
			report.Orphans = append(report.Orphans, identity)
		}
	}
	sort.Slice(report.Orphans, func(i, j int) bool {
		if report.Orphans[i].Username == report.Orphans[j].Username {
			return report.Orphans[i].Subject < report.Orphans[j].Subject
		}
		return report.Orphans[i].Username < report.Orphans[j].Username
	})

	return report, nil
}

func (r *userReconciler) reconcileUser(ctx context.Context, id int64) (string, error) {
	createdSubject := ""
	action := "updated"
	runErr := r.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		user, err := queries.GetUserForUpdate(ctx, id)
		if err != nil {
			return utils.UnexpectedDatabaseError(ctx, "lock user for reconciliation", err)
		}

		if user.IdentityIssuer != nil || user.ExternalSubject != nil {
			if err := requireManagedIdentity(user, r.issuer); err != nil {
				return err
			}
			return r.pushUser(ctx, user)
		}

		action = "provisioned"
		createdSubject, err = r.identities.CreateIdentity(ctx, userProfile(user))
		if err != nil {
			return err
		}
		if err := r.identities.ReplaceRole(
			ctx, createdSubject, types.UserRole(user.Role),
		); err != nil {
			return err
		}
		if err := r.identities.SetEnabled(ctx, createdSubject, user.IsActive); err != nil {
			return err
		}

		_, err = queries.LinkUserExternalIdentity(ctx, sqlcgen.LinkUserExternalIdentityParams{
			IdentityIssuer:  &r.issuer,
			ExternalSubject: &createdSubject,
			ID:              user.ID,
		})
		if err != nil {
			return utils.UnexpectedDatabaseError(ctx, "link reconciled user", err)
		}
		return nil
	})
	if runErr == nil {
		return action, nil
	}

	if createdSubject != "" {
		if err := r.identities.DeleteIdentity(context.Background(), createdSubject); err != nil {
			return "", fmt.Errorf("provision compensation failed: %w", runErr)
		}
	}
	return "", runErr
}

func (r *userReconciler) pushUser(ctx context.Context, user sqlcgen.User) error {
	subject := *user.ExternalSubject
	if err := r.identities.UpdateProfile(ctx, subject, userProfile(user)); err != nil {
		return err
	}
	if err := r.identities.ReplaceRole(ctx, subject, types.UserRole(user.Role)); err != nil {
		return err
	}
	return r.identities.SetEnabled(ctx, subject, user.IsActive)
}

func (r *userReconciler) linkedSubjects(ctx context.Context) (map[string]struct{}, error) {
	users, err := r.queries.ListUsers(ctx, nil)
	if err != nil {
		return nil, utils.UnexpectedDatabaseError(ctx, "list linked users", err)
	}

	subjects := make(map[string]struct{}, len(users))
	for _, user := range users {
		if user.IdentityIssuer != nil && user.ExternalSubject != nil &&
			*user.IdentityIssuer == r.issuer {
			subjects[*user.ExternalSubject] = struct{}{}
		}
	}
	return subjects, nil
}

func reconciliationFailureReason(err error) string {
	switch {
	case errors.Is(err, types.ErrIdentityAdminConflict):
		return "keycloak_conflict"
	case errors.Is(err, types.ErrIdentityAdminNotFound):
		return "keycloak_identity_not_found"
	case errors.Is(err, types.ErrIdentityAdminUnavailable):
		return "keycloak_unavailable"
	case errors.Is(err, types.ErrIdentityAdminRejected):
		return "keycloak_rejected"
	case errors.Is(err, types.ErrUserIdentityUnlinked):
		return "unsupported_identity_link"
	default:
		return "reconciliation_failed"
	}
}
