package services

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	constraintActiveCheckout = "uq_checkouts_one_active_per_item"
	constraintCheckoutDue    = "ck_checkouts_due_after_checkout"

	checkoutHistorySource  = "checkout"
	checkoutCreatedAction  = "checkout.created"
	checkoutReturnedAction = "checkout.returned"
	checkoutEntityType     = "checkout"
)

type CheckoutService interface {
	Checkout(
		ctx context.Context,
		itemID int64,
		input types.CreateCheckoutRequest,
		metadata types.WorkflowMetadata,
	) (types.Checkout, error)
	Return(
		ctx context.Context,
		checkoutID int64,
		metadata types.WorkflowMetadata,
	) (types.Checkout, error)
	GetByID(ctx context.Context, checkoutID int64) (types.Checkout, error)
	List(
		ctx context.Context,
		pagination types.PaginationRequest,
	) ([]types.Checkout, types.PaginationMetadata, error)
	ListByItem(
		ctx context.Context,
		itemID int64,
		pagination types.PaginationRequest,
	) ([]types.Checkout, types.PaginationMetadata, error)
}

type CheckoutTransactionRunner interface {
	Run(ctx context.Context, fn func(sqlcgen.Querier) error) error
}

type checkoutService struct {
	queries      sqlcgen.Querier
	transactions CheckoutTransactionRunner
}

var _ CheckoutService = (*checkoutService)(nil)

func NewCheckoutService(
	queries sqlcgen.Querier,
	transactions CheckoutTransactionRunner,
) CheckoutService {
	return &checkoutService{
		queries:      queries,
		transactions: transactions,
	}
}

func (s *checkoutService) Checkout(
	ctx context.Context,
	itemID int64,
	input types.CreateCheckoutRequest,
	metadata types.WorkflowMetadata,
) (types.Checkout, error) {
	if err := ctx.Err(); err != nil {
		return types.Checkout{}, err
	}
	if !utils.IsValidID(itemID) {
		return types.Checkout{}, types.ErrInvalidInput
	}

	actor, err := checkoutActor(ctx)
	if err != nil {
		return types.Checkout{}, err
	}
	input, err = normalizeCheckoutInput(input)
	if err != nil {
		return types.Checkout{}, err
	}

	var checkout types.Checkout
	err = s.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		borrower, err := queries.GetCheckoutBorrowerForShare(ctx, input.BorrowerUserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return types.ErrBorrowerNotFound
		}
		if err != nil {
			return err
		}
		if !borrower.IsActive {
			return types.ErrBorrowerInactive
		}

		item, err := queries.GetItemForStatusUpdate(ctx, itemID)
		if errors.Is(err, pgx.ErrNoRows) {
			return types.ErrItemNotFound
		}
		if err != nil {
			return err
		}
		if item.Status != types.ItemStatusAvailable {
			return types.ErrItemNotAvailable
		}

		transactionTime, err := queries.GetTransactionTimestamp(ctx)
		if err != nil {
			return err
		}
		if input.DueAt != nil && !input.DueAt.After(transactionTime.Time) {
			return types.ErrInvalidCheckoutDueAt
		}

		row, err := queries.CreateCheckout(ctx, sqlcgen.CreateCheckoutParams{
			ItemID:          itemID,
			BorrowerUserID:  borrower.ID,
			CreatedByUserID: actor.User.ID,
			DueAt:           checkoutDueAt(input.DueAt),
			Notes:           input.Notes,
		})
		if err != nil {
			return err
		}

		if _, err := queries.SetItemWorkflowStatus(ctx, sqlcgen.SetItemWorkflowStatusParams{
			NewStatus:      types.ItemStatusCheckedOut,
			ID:             itemID,
			ExpectedStatus: types.ItemStatusAvailable,
		}); errors.Is(err, pgx.ErrNoRows) {
			return types.ErrItemNotAvailable
		} else if err != nil {
			return err
		}

		if err := queries.RecordItemStatusHistory(ctx, sqlcgen.RecordItemStatusHistoryParams{
			ItemID:          itemID,
			ChangedByUserID: pointer(actor.User.ID),
			PreviousStatus:  pointer(types.ItemStatusAvailable),
			NewStatus:       types.ItemStatusCheckedOut,
			Reason:          pointer("item checked out"),
			SourceType:      pointer(checkoutHistorySource),
			SourceID:        pointer(row.ID),
		}); err != nil {
			return err
		}

		beforeData, afterData, err := itemStatusAuditData(
			types.ItemStatusAvailable,
			types.ItemStatusCheckedOut,
		)
		if err != nil {
			return err
		}

		if err := queries.RecordAuditEvent(ctx, sqlcgen.RecordAuditEventParams{
			ActorUserID:      pointer(actor.User.ID),
			Action:           checkoutCreatedAction,
			EntityType:       checkoutEntityType,
			EntityIdentifier: strconv.FormatInt(row.ID, 10),
			RequestID:        strings.TrimSpace(metadata.RequestID),
			BeforeData:       beforeData,
			AfterData:        afterData,
		}); err != nil {
			return err
		}

		checkout = checkoutFromRow(row)
		return nil
	})
	if err != nil {
		return types.Checkout{}, mapCheckoutWorkflowError(ctx, "checkout item", err)
	}

	return checkout, nil
}

func (s *checkoutService) Return(
	ctx context.Context,
	checkoutID int64,
	metadata types.WorkflowMetadata,
) (types.Checkout, error) {
	if err := ctx.Err(); err != nil {
		return types.Checkout{}, err
	}
	if !utils.IsValidID(checkoutID) {
		return types.Checkout{}, types.ErrInvalidCheckoutID
	}

	actor, err := checkoutActor(ctx)
	if err != nil {
		return types.Checkout{}, err
	}

	var checkout types.Checkout
	err = s.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		current, err := queries.GetCheckoutForUpdate(ctx, checkoutID)
		if errors.Is(err, pgx.ErrNoRows) {
			return types.ErrCheckoutNotFound
		}
		if err != nil {
			return err
		}
		if current.ReturnedAt.Valid {
			return types.ErrCheckoutReturned
		}

		item, err := queries.GetItemForStatusUpdate(ctx, current.ItemID)
		if errors.Is(err, pgx.ErrNoRows) {
			return types.ErrCheckoutStateConflict
		}
		if err != nil {
			return err
		}
		if item.Status != types.ItemStatusCheckedOut {
			return types.ErrCheckoutStateConflict
		}

		row, err := queries.ReturnCheckout(ctx, sqlcgen.ReturnCheckoutParams{
			ReturnedToUserID: pointer(actor.User.ID),
			ID:               checkoutID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return types.ErrCheckoutReturned
		}
		if err != nil {
			return err
		}

		if _, err := queries.SetItemWorkflowStatus(ctx, sqlcgen.SetItemWorkflowStatusParams{
			NewStatus:      types.ItemStatusAvailable,
			ID:             current.ItemID,
			ExpectedStatus: types.ItemStatusCheckedOut,
		}); errors.Is(err, pgx.ErrNoRows) {
			return types.ErrCheckoutStateConflict
		} else if err != nil {
			return err
		}

		if err := queries.RecordItemStatusHistory(ctx, sqlcgen.RecordItemStatusHistoryParams{
			ItemID:          current.ItemID,
			ChangedByUserID: pointer(actor.User.ID),
			PreviousStatus:  pointer(types.ItemStatusCheckedOut),
			NewStatus:       types.ItemStatusAvailable,
			Reason:          pointer("item returned"),
			SourceType:      pointer(checkoutHistorySource),
			SourceID:        pointer(checkoutID),
		}); err != nil {
			return err
		}

		beforeData, afterData, err := itemStatusAuditData(
			types.ItemStatusCheckedOut,
			types.ItemStatusAvailable,
		)
		if err != nil {
			return err
		}
		if err := queries.RecordAuditEvent(ctx, sqlcgen.RecordAuditEventParams{
			ActorUserID:      pointer(actor.User.ID),
			Action:           checkoutReturnedAction,
			EntityType:       checkoutEntityType,
			EntityIdentifier: strconv.FormatInt(checkoutID, 10),
			RequestID:        strings.TrimSpace(metadata.RequestID),
			BeforeData:       beforeData,
			AfterData:        afterData,
		}); err != nil {
			return err
		}

		checkout = checkoutFromRow(row)
		return nil
	})
	if err != nil {
		return types.Checkout{}, mapCheckoutWorkflowError(ctx, "return checkout", err)
	}

	return checkout, nil
}

func (s *checkoutService) GetByID(
	ctx context.Context,
	checkoutID int64,
) (types.Checkout, error) {
	if err := ctx.Err(); err != nil {
		return types.Checkout{}, err
	}
	if !utils.IsValidID(checkoutID) {
		return types.Checkout{}, types.ErrInvalidCheckoutID
	}

	row, err := s.queries.GetCheckout(ctx, checkoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.Checkout{}, types.ErrCheckoutNotFound
	}
	if err != nil {
		return types.Checkout{}, utils.UnexpectedDatabaseError(ctx, "get checkout", err)
	}

	return checkoutFromRow(row), nil
}

func (s *checkoutService) List(
	ctx context.Context,
	pagination types.PaginationRequest,
) ([]types.Checkout, types.PaginationMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.PaginationMetadata{}, err
	}
	if err := validateCheckoutPagination(pagination); err != nil {
		return nil, types.PaginationMetadata{}, err
	}

	total, err := s.queries.CountCheckouts(ctx)
	if err != nil {
		return nil, types.PaginationMetadata{},
			utils.UnexpectedDatabaseError(ctx, "count checkouts", err)
	}

	rows, err := s.queries.ListCheckouts(ctx, sqlcgen.ListCheckoutsParams{
		PageOffset: pagination.Offset,
		PageLimit:  pagination.Limit,
	})
	if err != nil {
		return nil, types.PaginationMetadata{},
			utils.UnexpectedDatabaseError(ctx, "list checkouts", err)
	}

	checkouts := checkoutsFromRows(rows)
	return checkouts, paginationMetadata(pagination, len(checkouts), total), nil
}

func (s *checkoutService) ListByItem(
	ctx context.Context,
	itemID int64,
	pagination types.PaginationRequest,
) ([]types.Checkout, types.PaginationMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.PaginationMetadata{}, err
	}
	if !utils.IsValidID(itemID) {
		return nil, types.PaginationMetadata{}, types.ErrInvalidInput
	}
	if err := validateCheckoutPagination(pagination); err != nil {
		return nil, types.PaginationMetadata{}, err
	}

	if _, err := s.queries.GetItem(ctx, itemID); errors.Is(err, pgx.ErrNoRows) {
		return nil, types.PaginationMetadata{}, types.ErrItemNotFound
	} else if err != nil {
		return nil, types.PaginationMetadata{},
			utils.UnexpectedDatabaseError(ctx, "get checkout item", err)
	}

	total, err := s.queries.CountItemCheckouts(ctx, itemID)
	if err != nil {
		return nil, types.PaginationMetadata{},
			utils.UnexpectedDatabaseError(ctx, "count item checkouts", err)
	}

	rows, err := s.queries.ListItemCheckouts(ctx, sqlcgen.ListItemCheckoutsParams{
		ItemID:     itemID,
		PageOffset: pagination.Offset,
		PageLimit:  pagination.Limit,
	})
	if err != nil {
		return nil, types.PaginationMetadata{},
			utils.UnexpectedDatabaseError(ctx, "list item checkouts", err)
	}

	checkouts := checkoutsFromRows(rows)
	return checkouts, paginationMetadata(pagination, len(checkouts), total), nil
}

func normalizeCheckoutInput(
	input types.CreateCheckoutRequest,
) (types.CreateCheckoutRequest, error) {
	if !utils.IsValidID(input.BorrowerUserID) {
		return types.CreateCheckoutRequest{}, types.ErrInvalidBorrowerID
	}

	input.Notes = strings.TrimSpace(input.Notes)
	if input.DueAt != nil {
		dueAt := input.DueAt.UTC()
		input.DueAt = &dueAt
	}

	return input, nil
}

func validateCheckoutPagination(pagination types.PaginationRequest) error {
	if pagination.Limit < 1 || pagination.Limit > 100 || pagination.Offset < 0 {
		return types.ErrInvalidPagination
	}
	return nil
}

func checkoutActor(ctx context.Context) (types.Actor, error) {
	actor, ok := types.ActorFromContext(ctx)
	if !ok {
		return types.Actor{}, types.ErrActorRequired
	}
	return actor, nil
}

func checkoutDueAt(dueAt *time.Time) pgtype.Timestamptz {
	if dueAt == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: dueAt.UTC(), Valid: true}
}

func checkoutFromRow(row sqlcgen.Checkout) types.Checkout {
	return types.Checkout{
		ID:               row.ID,
		ItemID:           row.ItemID,
		BorrowerUserID:   row.BorrowerUserID,
		CreatedByUserID:  row.CreatedByUserID,
		ReturnedToUserID: row.ReturnedToUserID,
		CheckedOutAt:     row.CheckedOutAt.Time,
		DueAt:            nullableTime(row.DueAt),
		ReturnedAt:       nullableTime(row.ReturnedAt),
		Notes:            row.Notes,
	}
}

func checkoutsFromRows(rows []sqlcgen.Checkout) []types.Checkout {
	checkouts := make([]types.Checkout, 0, len(rows))
	for _, row := range rows {
		checkouts = append(checkouts, checkoutFromRow(row))
	}
	return checkouts
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return pointer(value.Time)
}

func paginationMetadata(
	pagination types.PaginationRequest,
	count int,
	total int64,
) types.PaginationMetadata {
	return types.PaginationMetadata{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
		Count:  count,
		Total:  total,
	}
}

func itemStatusAuditData(previousStatus, newStatus string) ([]byte, []byte, error) {
	beforeData, err := json.Marshal(struct {
		ItemStatus string `json:"item_status"`
	}{
		ItemStatus: previousStatus,
	})
	if err != nil {
		return nil, nil, err
	}

	afterData, err := json.Marshal(struct {
		ItemStatus string `json:"item_status"`
	}{
		ItemStatus: newStatus,
	})
	if err != nil {
		return nil, nil, err
	}

	return beforeData, afterData, nil
}

func mapCheckoutWorkflowError(ctx context.Context, operation string, err error) error {
	switch {
	case errors.Is(err, types.ErrActorRequired),
		errors.Is(err, types.ErrInvalidBorrowerID),
		errors.Is(err, types.ErrBorrowerNotFound),
		errors.Is(err, types.ErrBorrowerInactive),
		errors.Is(err, types.ErrInvalidCheckoutDueAt),
		errors.Is(err, types.ErrItemNotFound),
		errors.Is(err, types.ErrItemNotAvailable),
		errors.Is(err, types.ErrCheckoutNotFound),
		errors.Is(err, types.ErrCheckoutReturned),
		errors.Is(err, types.ErrCheckoutStateConflict):
		return err
	}

	pgError, ok := utils.PostgresError(err)
	if ok {
		switch {
		case pgError.Code == utils.PostgresUniqueViolation &&
			pgError.ConstraintName == constraintActiveCheckout:
			return types.ErrItemNotAvailable
		case pgError.Code == utils.PostgresCheckViolation &&
			pgError.ConstraintName == constraintCheckoutDue:
			return types.ErrInvalidCheckoutDueAt
		}
	}

	return utils.UnexpectedDatabaseError(ctx, operation, err)
}

func pointer[T any](value T) *T {
	return &value
}
