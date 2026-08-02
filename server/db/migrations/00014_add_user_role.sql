-- +goose Up
ALTER TABLE users
    ADD COLUMN role TEXT;

UPDATE users
SET role = CASE lower(btrim(username))
    WHEN 'equipment.admin' THEN 'inventory_admin'
    WHEN 'audit.viewer' THEN 'auditor'
    ELSE 'employee'
END;

ALTER TABLE users
    ALTER COLUMN role SET NOT NULL,
    ADD CONSTRAINT ck_users_role
        CHECK (role IN ('employee', 'inventory_admin', 'auditor'));

-- +goose Down
ALTER TABLE users
    DROP CONSTRAINT ck_users_role,
    DROP COLUMN role;
