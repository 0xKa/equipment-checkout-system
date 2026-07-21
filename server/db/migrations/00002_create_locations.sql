-- +goose Up
CREATE TABLE locations (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    parent_location_id BIGINT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_locations PRIMARY KEY (id),
    CONSTRAINT fk_locations_parent_location
        FOREIGN KEY (parent_location_id) REFERENCES locations (id) ON DELETE RESTRICT,
    CONSTRAINT uq_locations_code UNIQUE (code),
    CONSTRAINT ck_locations_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT ck_locations_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_locations_not_own_parent CHECK (parent_location_id IS NULL OR parent_location_id <> id)
);

CREATE INDEX idx_locations_parent_location_id ON locations (parent_location_id);

-- +goose Down
DROP TABLE locations;
