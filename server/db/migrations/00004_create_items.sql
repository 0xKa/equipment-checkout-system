-- +goose Up
CREATE TABLE items (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    category_id BIGINT,
    location_id BIGINT,
    asset_tag TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    serial_number TEXT,
    status TEXT NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_items PRIMARY KEY (id),
    CONSTRAINT fk_items_category
        FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE RESTRICT,
    CONSTRAINT fk_items_location
        FOREIGN KEY (location_id) REFERENCES locations (id) ON DELETE RESTRICT,
    CONSTRAINT uq_items_asset_tag UNIQUE (asset_tag),
    CONSTRAINT ck_items_asset_tag_not_blank CHECK (btrim(asset_tag) <> ''),
    CONSTRAINT ck_items_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_items_serial_number_not_blank
        CHECK (serial_number IS NULL OR btrim(serial_number) <> ''),
    CONSTRAINT ck_items_status
        CHECK (status IN ('available', 'checked_out', 'maintenance', 'retired'))
);

CREATE INDEX idx_items_category_id ON items (category_id);
CREATE INDEX idx_items_location_id ON items (location_id);
CREATE UNIQUE INDEX uq_items_serial_number
    ON items (serial_number)
    WHERE serial_number IS NOT NULL;

-- +goose Down
DROP TABLE items;
