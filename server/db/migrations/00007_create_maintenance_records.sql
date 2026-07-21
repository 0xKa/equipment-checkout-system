-- +goose Up
CREATE TABLE maintenance_records (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    item_id BIGINT NOT NULL,
    opened_by_user_id BIGINT NOT NULL,
    assigned_to_user_id BIGINT,
    status TEXT NOT NULL DEFAULT 'open',
    reason TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_maintenance_records PRIMARY KEY (id),
    CONSTRAINT fk_maintenance_records_item
        FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE RESTRICT,
    CONSTRAINT fk_maintenance_records_opened_by_user
        FOREIGN KEY (opened_by_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_maintenance_records_assigned_to_user
        FOREIGN KEY (assigned_to_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_maintenance_records_status
        CHECK (status IN ('open', 'in_progress', 'completed', 'cancelled')),
    CONSTRAINT ck_maintenance_records_reason_not_blank CHECK (btrim(reason) <> ''),
    CONSTRAINT ck_maintenance_records_completion_time
        CHECK (completed_at IS NULL OR started_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX idx_maintenance_records_item_id ON maintenance_records (item_id);
CREATE INDEX idx_maintenance_records_opened_by_user_id ON maintenance_records (opened_by_user_id);
CREATE INDEX idx_maintenance_records_assigned_to_user_id ON maintenance_records (assigned_to_user_id);

-- +goose Down
DROP TABLE maintenance_records;
