-- +goose Up
CREATE TABLE reservations (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    item_id BIGINT NOT NULL,
    requested_by_user_id BIGINT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_reservations PRIMARY KEY (id),
    CONSTRAINT fk_reservations_item
        FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE RESTRICT,
    CONSTRAINT fk_reservations_requested_by_user
        FOREIGN KEY (requested_by_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_reservations_valid_time_range CHECK (ends_at > starts_at),
    CONSTRAINT ck_reservations_status
        CHECK (status IN ('pending', 'approved', 'cancelled', 'fulfilled', 'expired'))
);

CREATE INDEX idx_reservations_item_id ON reservations (item_id);
CREATE INDEX idx_reservations_requested_by_user_id ON reservations (requested_by_user_id);

-- +goose Down
DROP TABLE reservations;
