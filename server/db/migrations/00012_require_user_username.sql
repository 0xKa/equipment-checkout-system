-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE username IS NULL) THEN
        RAISE EXCEPTION 'cannot require users.username while rows with null usernames exist';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE users
    DROP CONSTRAINT ck_users_identifier_present,
    ALTER COLUMN username SET NOT NULL;

DROP INDEX uq_users_username_normalized;

CREATE UNIQUE INDEX uq_users_username_normalized
    ON users (lower(btrim(username)));

-- +goose Down
DROP INDEX uq_users_username_normalized;

CREATE UNIQUE INDEX uq_users_username_normalized
    ON users (lower(btrim(username)))
    WHERE username IS NOT NULL;

ALTER TABLE users
    ALTER COLUMN username DROP NOT NULL,
    ADD CONSTRAINT ck_users_identifier_present
        CHECK (username IS NOT NULL OR email IS NOT NULL);
