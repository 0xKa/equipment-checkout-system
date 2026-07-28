-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    display_name
) VALUES (
    sqlc.arg(username),
    sqlc.narg(email),
    sqlc.arg(display_name)
)
RETURNING
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject;

-- name: CreateUserWithExternalIdentity :one
INSERT INTO users (
    username,
    email,
    display_name,
    identity_issuer,
    external_subject
) VALUES (
    sqlc.arg(username),
    sqlc.narg(email),
    sqlc.arg(display_name),
    sqlc.arg(identity_issuer),
    sqlc.arg(external_subject)
)
RETURNING
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject;

-- name: ListUsers :many
SELECT
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject
FROM users
WHERE sqlc.narg(is_active)::boolean IS NULL
   OR is_active = sqlc.narg(is_active)
ORDER BY id ASC;

-- name: GetUser :one
SELECT
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject
FROM users
WHERE id = sqlc.arg(id);

-- name: GetUserByExternalIdentity :one
SELECT
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject
FROM users
WHERE identity_issuer = sqlc.arg(identity_issuer)
  AND external_subject = sqlc.arg(external_subject);

-- name: UpdateUser :one
UPDATE users
SET
    username = sqlc.arg(username),
    email = sqlc.narg(email),
    display_name = sqlc.arg(display_name),
    updated_at = GREATEST(statement_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
RETURNING
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject;

-- name: SetUserActive :one
UPDATE users
SET
    is_active = sqlc.arg(is_active),
    updated_at = GREATEST(statement_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
RETURNING
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject;

-- name: UsernameExists :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE lower(btrim(username)) = lower(btrim(sqlc.arg(username)))
      AND id <> sqlc.arg(excluded_id)
);

-- name: UserEmailExists :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE email IS NOT NULL
      AND lower(btrim(email)) = lower(btrim(sqlc.arg(email)))
      AND id <> sqlc.arg(excluded_id)
);
