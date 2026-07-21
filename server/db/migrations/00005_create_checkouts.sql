-- +goose Up
CREATE TABLE checkouts (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    item_id BIGINT NOT NULL,
    borrower_user_id BIGINT NOT NULL,
    created_by_user_id BIGINT NOT NULL,
    returned_to_user_id BIGINT,
    checked_out_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_at TIMESTAMPTZ,
    returned_at TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    CONSTRAINT pk_checkouts PRIMARY KEY (id),
    CONSTRAINT fk_checkouts_item
        FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE RESTRICT,
    CONSTRAINT fk_checkouts_borrower_user
        FOREIGN KEY (borrower_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_checkouts_created_by_user
        FOREIGN KEY (created_by_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_checkouts_returned_to_user
        FOREIGN KEY (returned_to_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_checkouts_due_after_checkout
        CHECK (due_at IS NULL OR due_at > checked_out_at),
    CONSTRAINT ck_checkouts_return_after_checkout
        CHECK (returned_at IS NULL OR returned_at > checked_out_at),
    CONSTRAINT ck_checkouts_return_fields_consistent
        CHECK ((returned_at IS NULL) = (returned_to_user_id IS NULL))
);

CREATE INDEX idx_checkouts_item_id ON checkouts (item_id);
CREATE INDEX idx_checkouts_borrower_user_id ON checkouts (borrower_user_id);
CREATE INDEX idx_checkouts_created_by_user_id ON checkouts (created_by_user_id);
CREATE INDEX idx_checkouts_returned_to_user_id ON checkouts (returned_to_user_id);
CREATE UNIQUE INDEX uq_checkouts_one_active_per_item
    ON checkouts (item_id)
    WHERE returned_at IS NULL;

-- +goose Down
DROP TABLE checkouts;
