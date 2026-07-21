-- name: CreateCategory :one
INSERT INTO categories (
    name,
    description
) VALUES (
    sqlc.arg(name),
    sqlc.arg(description)
)
RETURNING
    id,
    name,
    description,
    created_at,
    updated_at;

-- name: ListCategories :many
SELECT
    id,
    name,
    description,
    created_at,
    updated_at
FROM categories
ORDER BY id ASC;

-- name: GetCategory :one
SELECT
    id,
    name,
    description,
    created_at,
    updated_at
FROM categories
WHERE id = sqlc.arg(id);

-- name: UpdateCategory :one
UPDATE categories
SET
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    updated_at = GREATEST(statement_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
RETURNING
    id,
    name,
    description,
    created_at,
    updated_at;

-- name: DeleteCategory :one
DELETE FROM categories
WHERE id = sqlc.arg(id)
RETURNING id;

-- name: CategoryNameExists :one
SELECT EXISTS (
    SELECT 1
    FROM categories
    WHERE name = sqlc.arg(name)
      AND id <> sqlc.arg(excluded_id)
);

-- name: CategoryExists :one
SELECT EXISTS (
    SELECT 1
    FROM categories
    WHERE id = sqlc.arg(id)
);
