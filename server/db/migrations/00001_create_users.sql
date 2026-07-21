-- +goose Up
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    username TEXT,
    email TEXT,
    display_name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT ck_users_username_not_blank CHECK (username IS NULL OR btrim(username) <> ''),
    CONSTRAINT ck_users_email_not_blank CHECK (email IS NULL OR btrim(email) <> ''),
    CONSTRAINT ck_users_display_name_not_blank CHECK (display_name IS NULL OR btrim(display_name) <> '')
);

-- +goose Down
DROP TABLE users;
