-- +goose Up
CREATE TABLE condition_reports (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    item_id BIGINT NOT NULL,
    reported_by_user_id BIGINT NOT NULL,
    checkout_id BIGINT,
    maintenance_record_id BIGINT,
    report_stage TEXT NOT NULL,
    condition_rating TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_condition_reports PRIMARY KEY (id),
    CONSTRAINT fk_condition_reports_item
        FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE RESTRICT,
    CONSTRAINT fk_condition_reports_reported_by_user
        FOREIGN KEY (reported_by_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_condition_reports_checkout
        FOREIGN KEY (checkout_id) REFERENCES checkouts (id) ON DELETE RESTRICT,
    CONSTRAINT fk_condition_reports_maintenance_record
        FOREIGN KEY (maintenance_record_id) REFERENCES maintenance_records (id) ON DELETE RESTRICT,
    CONSTRAINT ck_condition_reports_stage
        CHECK (report_stage IN ('checkout', 'return', 'maintenance', 'inspection')),
    CONSTRAINT ck_condition_reports_rating
        CHECK (condition_rating IN ('excellent', 'good', 'fair', 'poor', 'damaged')),
    CONSTRAINT ck_condition_reports_at_most_one_context
        CHECK (checkout_id IS NULL OR maintenance_record_id IS NULL)
);

CREATE INDEX idx_condition_reports_item_id ON condition_reports (item_id);
CREATE INDEX idx_condition_reports_reported_by_user_id ON condition_reports (reported_by_user_id);
CREATE INDEX idx_condition_reports_checkout_id ON condition_reports (checkout_id);
CREATE INDEX idx_condition_reports_maintenance_record_id ON condition_reports (maintenance_record_id);

-- +goose Down
DROP TABLE condition_reports;
