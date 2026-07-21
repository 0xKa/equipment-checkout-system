-- +goose Up
CREATE TABLE audit_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    actor_user_id BIGINT,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_identifier TEXT NOT NULL,
    request_id TEXT,
    source_ip INET,
    before_data JSONB,
    after_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_audit_events PRIMARY KEY (id),
    CONSTRAINT fk_audit_events_actor_user
        FOREIGN KEY (actor_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_audit_events_action_not_blank CHECK (btrim(action) <> ''),
    CONSTRAINT ck_audit_events_entity_type_not_blank CHECK (btrim(entity_type) <> ''),
    CONSTRAINT ck_audit_events_entity_identifier_not_blank CHECK (btrim(entity_identifier) <> ''),
    CONSTRAINT ck_audit_events_request_id_not_blank
        CHECK (request_id IS NULL OR btrim(request_id) <> '')
);

CREATE INDEX idx_audit_events_actor_user_id ON audit_events (actor_user_id);
CREATE INDEX idx_audit_events_entity ON audit_events (entity_type, entity_identifier);
CREATE INDEX idx_audit_events_created_at ON audit_events (created_at);
CREATE INDEX idx_audit_events_request_id ON audit_events (request_id) WHERE request_id IS NOT NULL;

-- +goose Down
DROP TABLE audit_events;
