-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    display_name,
    role,
    identity_issuer,
    external_subject
) VALUES (
    sqlc.arg(username),
    sqlc.narg(email),
    sqlc.arg(display_name),
    sqlc.arg(role),
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
    external_subject,
    role;

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
    external_subject,
    role
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
    external_subject,
    role
FROM users
WHERE id = sqlc.arg(id);

-- name: GetUserForUpdate :one
SELECT
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject,
    role
FROM users
WHERE id = sqlc.arg(id)
FOR UPDATE;

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
    external_subject,
    role
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
    external_subject,
    role;

-- name: SetUserRole :one
UPDATE users
SET
    role = sqlc.arg(role),
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
    external_subject,
    role;

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
    external_subject,
    role;

-- name: LinkUserExternalIdentity :one
UPDATE users
SET
    identity_issuer = sqlc.arg(identity_issuer),
    external_subject = sqlc.arg(external_subject),
    updated_at = GREATEST(statement_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
  AND identity_issuer IS NULL
  AND external_subject IS NULL
RETURNING
    id,
    username,
    email,
    display_name,
    is_active,
    created_at,
    updated_at,
    identity_issuer,
    external_subject,
    role;

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
