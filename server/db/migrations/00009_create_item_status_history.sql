-- +goose Up
CREATE TABLE item_status_history (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    item_id BIGINT NOT NULL,
    changed_by_user_id BIGINT,
    previous_status TEXT,
    new_status TEXT NOT NULL,
    reason TEXT,
    source_type TEXT,
    source_id BIGINT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_item_status_history PRIMARY KEY (id),
    CONSTRAINT fk_item_status_history_item
        FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE RESTRICT,
    CONSTRAINT fk_item_status_history_changed_by_user
        FOREIGN KEY (changed_by_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_item_status_history_previous_status
        CHECK (
            previous_status IS NULL
            OR previous_status IN ('available', 'checked_out', 'maintenance', 'retired')
        ),
    CONSTRAINT ck_item_status_history_new_status
        CHECK (new_status IN ('available', 'checked_out', 'maintenance', 'retired')),
    CONSTRAINT ck_item_status_history_status_changed
        CHECK (previous_status IS NULL OR previous_status <> new_status),
    CONSTRAINT ck_item_status_history_reason_not_blank
        CHECK (reason IS NULL OR btrim(reason) <> ''),
    CONSTRAINT ck_item_status_history_source_type_not_blank
        CHECK (source_type IS NULL OR btrim(source_type) <> ''),
    CONSTRAINT ck_item_status_history_source_fields_consistent
        CHECK ((source_type IS NULL) = (source_id IS NULL)),
    CONSTRAINT ck_item_status_history_source_id_positive
        CHECK (source_id IS NULL OR source_id > 0)
);

CREATE INDEX idx_item_status_history_item_id ON item_status_history (item_id);
CREATE INDEX idx_item_status_history_changed_by_user_id ON item_status_history (changed_by_user_id);
CREATE INDEX idx_item_status_history_changed_at ON item_status_history (changed_at);

-- +goose Down
DROP TABLE item_status_history;
