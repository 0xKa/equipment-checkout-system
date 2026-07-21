-- name: CreateItem :one
INSERT INTO items (
    category_id,
    asset_tag,
    name,
    description,
    serial_number
) VALUES (
    sqlc.narg(category_id),
    sqlc.arg(asset_tag),
    sqlc.arg(name),
    sqlc.arg(description),
    sqlc.narg(serial_number)
)
RETURNING
    id,
    category_id,
    location_id,
    asset_tag,
    name,
    description,
    serial_number,
    status,
    created_at,
    updated_at;

-- name: ListItems :many
SELECT
    id,
    category_id,
    location_id,
    asset_tag,
    name,
    description,
    serial_number,
    status,
    created_at,
    updated_at
FROM items
ORDER BY id ASC;

-- name: GetItem :one
SELECT
    id,
    category_id,
    location_id,
    asset_tag,
    name,
    description,
    serial_number,
    status,
    created_at,
    updated_at
FROM items
WHERE id = sqlc.arg(id);

-- name: UpdateItem :one
UPDATE items
SET
    category_id = sqlc.narg(category_id),
    asset_tag = sqlc.arg(asset_tag),
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    serial_number = sqlc.narg(serial_number),
    updated_at = GREATEST(statement_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
RETURNING
    id,
    category_id,
    location_id,
    asset_tag,
    name,
    description,
    serial_number,
    status,
    created_at,
    updated_at;

-- name: DeleteItem :one
DELETE FROM items
WHERE id = sqlc.arg(id)
RETURNING id;

-- name: AssetTagExists :one
SELECT EXISTS (
    SELECT 1
    FROM items
    WHERE asset_tag = sqlc.arg(asset_tag)
      AND id <> sqlc.arg(excluded_id)
);
