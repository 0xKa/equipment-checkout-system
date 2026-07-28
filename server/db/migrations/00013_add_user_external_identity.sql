-- +goose Up
ALTER TABLE users
    ADD COLUMN identity_issuer TEXT,
    ADD COLUMN external_subject TEXT,
    ADD CONSTRAINT ck_users_external_identity_pair
        CHECK (
            (identity_issuer IS NULL AND external_subject IS NULL)
            OR
            (identity_issuer IS NOT NULL AND external_subject IS NOT NULL)
        ),
    ADD CONSTRAINT ck_users_identity_issuer_not_blank
        CHECK (identity_issuer IS NULL OR btrim(identity_issuer) <> ''),
    ADD CONSTRAINT ck_users_external_subject_not_blank
        CHECK (external_subject IS NULL OR btrim(external_subject) <> '');

CREATE UNIQUE INDEX uq_users_external_identity
    ON users (identity_issuer, external_subject)
    WHERE identity_issuer IS NOT NULL
      AND external_subject IS NOT NULL;

-- +goose Down
DROP INDEX uq_users_external_identity;

ALTER TABLE users
    DROP CONSTRAINT ck_users_external_subject_not_blank,
    DROP CONSTRAINT ck_users_identity_issuer_not_blank,
    DROP CONSTRAINT ck_users_external_identity_pair,
    DROP COLUMN external_subject,
    DROP COLUMN identity_issuer;
