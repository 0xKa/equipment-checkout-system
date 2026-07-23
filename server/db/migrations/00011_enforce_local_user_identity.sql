-- +goose Up
ALTER TABLE users
    ALTER COLUMN display_name SET NOT NULL,
    ADD CONSTRAINT ck_users_identifier_present
        CHECK (username IS NOT NULL OR email IS NOT NULL);

CREATE UNIQUE INDEX uq_users_username_normalized
    ON users (lower(btrim(username)))
    WHERE username IS NOT NULL;

CREATE UNIQUE INDEX uq_users_email_normalized
    ON users (lower(btrim(email)))
    WHERE email IS NOT NULL;

-- +goose Down
DROP INDEX uq_users_email_normalized;
DROP INDEX uq_users_username_normalized;

ALTER TABLE users
    DROP CONSTRAINT ck_users_identifier_present,
    ALTER COLUMN display_name DROP NOT NULL;
