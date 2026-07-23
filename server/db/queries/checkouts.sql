-- name: GetCheckout :one
SELECT
    id,
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes
FROM checkouts
WHERE id = sqlc.arg(id);

-- name: CountCheckouts :one
SELECT count(*)::bigint
FROM checkouts;

-- name: ListCheckouts :many
SELECT
    id,
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes
FROM checkouts
ORDER BY id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountItemCheckouts :one
SELECT count(*)::bigint
FROM checkouts
WHERE item_id = sqlc.arg(item_id);

-- name: ListItemCheckouts :many
SELECT
    id,
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes
FROM checkouts
WHERE item_id = sqlc.arg(item_id)
ORDER BY id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: GetCheckoutBorrowerForShare :one
SELECT
    id,
    is_active
FROM users
WHERE id = sqlc.arg(id)
FOR SHARE;

-- name: GetItemForStatusUpdate :one
SELECT
    id,
    status
FROM items
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: GetTransactionTimestamp :one
SELECT transaction_timestamp()::timestamptz;

-- name: CreateCheckout :one
INSERT INTO checkouts (
    item_id,
    borrower_user_id,
    created_by_user_id,
    due_at,
    notes
) VALUES (
    sqlc.arg(item_id),
    sqlc.arg(borrower_user_id),
    sqlc.arg(created_by_user_id),
    sqlc.narg(due_at),
    sqlc.arg(notes)
)
RETURNING
    id,
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes;

-- name: SetItemWorkflowStatus :one
UPDATE items
SET
    status = sqlc.arg(new_status),
    updated_at = GREATEST(statement_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
  AND status = sqlc.arg(expected_status)
RETURNING id;

-- name: RecordItemStatusHistory :exec
INSERT INTO item_status_history (
    item_id,
    changed_by_user_id,
    previous_status,
    new_status,
    reason,
    source_type,
    source_id
) VALUES (
    sqlc.arg(item_id),
    sqlc.arg(changed_by_user_id),
    sqlc.arg(previous_status),
    sqlc.arg(new_status),
    sqlc.arg(reason),
    sqlc.arg(source_type),
    sqlc.arg(source_id)
);

-- name: RecordAuditEvent :exec
INSERT INTO audit_events (
    actor_user_id,
    action,
    entity_type,
    entity_identifier,
    request_id,
    before_data,
    after_data
) VALUES (
    sqlc.arg(actor_user_id),
    sqlc.arg(action),
    sqlc.arg(entity_type),
    sqlc.arg(entity_identifier),
    NULLIF(sqlc.arg(request_id)::text, ''),
    sqlc.narg(before_data),
    sqlc.narg(after_data)
);

-- name: GetCheckoutForUpdate :one
SELECT
    id,
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes
FROM checkouts
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: ReturnCheckout :one
UPDATE checkouts
SET
    returned_at = GREATEST(statement_timestamp(), checked_out_at + INTERVAL '1 microsecond'),
    returned_to_user_id = sqlc.arg(returned_to_user_id)
WHERE id = sqlc.arg(id)
  AND returned_at IS NULL
RETURNING
    id,
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes;
