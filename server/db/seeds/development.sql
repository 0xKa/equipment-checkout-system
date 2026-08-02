-- Repeatable sample data for local development only.
-- Apply the schema migrations before running this script.

INSERT INTO users (username, email, display_name, role)
SELECT 'equipment.admin', 'admin@example.test', 'Equipment Administrator', 'inventory_admin'
WHERE NOT EXISTS (
    SELECT 1
    FROM users
    WHERE lower(btrim(username)) = 'equipment.admin'
);

INSERT INTO users (username, email, display_name, role)
SELECT 'sample.borrower', 'borrower@example.test', 'Sample Borrower', 'employee'
WHERE NOT EXISTS (
    SELECT 1
    FROM users
    WHERE lower(btrim(username)) = 'sample.borrower'
);

INSERT INTO users (username, email, display_name, role)
SELECT 'audit.viewer', 'audit@example.test', 'Audit Viewer', 'auditor'
WHERE NOT EXISTS (
    SELECT 1
    FROM users
    WHERE lower(btrim(username)) = 'audit.viewer'
);

INSERT INTO users (username, email, display_name, role)
SELECT 'maintenance.tech', 'technician@example.test', 'Maintenance Technician', 'employee'
WHERE NOT EXISTS (
    SELECT 1
    FROM users
    WHERE lower(btrim(username)) = 'maintenance.tech'
);

-- Link only the exact intended development profiles to canonical Keycloak
-- identities. A username-only match is intentionally insufficient; any other
-- local row remains unlinked for explicit reconciliation or conflict review.
UPDATE users
SET
    identity_issuer = development_identities.identity_issuer,
    external_subject = development_identities.external_subject,
    updated_at = GREATEST(statement_timestamp(), users.updated_at + INTERVAL '1 microsecond')
FROM (
    VALUES
        (
            'equipment.admin',
            'admin@example.test',
            'http://localhost:8081/realms/equipment',
            'e6549747-b961-4b7f-8b7d-dd894bca6d75'
        ),
        (
            'sample.borrower',
            'borrower@example.test',
            'http://localhost:8081/realms/equipment',
            '65b693e8-4199-452b-8277-ae9fd2264ac3'
        ),
        (
            'audit.viewer',
            'audit@example.test',
            'http://localhost:8081/realms/equipment',
            '16c37149-7558-454f-94c1-73ca1e337541'
        )
) AS development_identities(username, email, identity_issuer, external_subject)
WHERE users.username = development_identities.username
  AND users.email = development_identities.email
  AND (
      users.identity_issuer IS DISTINCT FROM development_identities.identity_issuer
      OR users.external_subject IS DISTINCT FROM development_identities.external_subject
  );

INSERT INTO locations (code, name, description)
VALUES ('HQ', 'Headquarters', 'Primary office location')
ON CONFLICT (code) DO NOTHING;

INSERT INTO locations (parent_location_id, code, name, description)
SELECT id, 'STORAGE-A', 'Storage Room A', 'Primary equipment storage room'
FROM locations
WHERE code = 'HQ'
ON CONFLICT (code) DO NOTHING;

INSERT INTO locations (parent_location_id, code, name, description)
SELECT id, 'OFFICE-101', 'Office 101', 'General staff workspace'
FROM locations
WHERE code = 'HQ'
ON CONFLICT (code) DO NOTHING;

INSERT INTO categories (name, description)
VALUES
    ('Laptop', 'Portable computers'),
    ('Monitor', 'External displays'),
    ('Accessory', 'Equipment accessories and peripherals')
ON CONFLICT (name) DO NOTHING;

INSERT INTO items (asset_tag, name, description, serial_number, category_id, location_id)
SELECT
    'EQ-LAP-001',
    'Development Laptop',
    'Sample laptop for local development',
    'DEV-LAPTOP-001',
    categories.id,
    locations.id
FROM categories
CROSS JOIN locations
WHERE categories.name = 'Laptop'
  AND locations.code = 'OFFICE-101'
ON CONFLICT (asset_tag) DO NOTHING;

INSERT INTO items (asset_tag, name, description, serial_number, category_id, location_id)
SELECT
    'EQ-MON-001',
    'Development Monitor',
    'Sample monitor for local development',
    'DEV-MONITOR-001',
    categories.id,
    locations.id
FROM categories
CROSS JOIN locations
WHERE categories.name = 'Monitor'
  AND locations.code = 'STORAGE-A'
ON CONFLICT (asset_tag) DO NOTHING;

INSERT INTO items (asset_tag, name, description, serial_number, category_id, location_id)
SELECT
    'EQ-ACC-001',
    'USB-C Dock',
    'Sample accessory for local development',
    'DEV-DOCK-001',
    categories.id,
    locations.id
FROM categories
CROSS JOIN locations
WHERE categories.name = 'Accessory'
  AND locations.code = 'STORAGE-A'
ON CONFLICT (asset_tag) DO NOTHING;

INSERT INTO checkouts (
    item_id,
    borrower_user_id,
    created_by_user_id,
    returned_to_user_id,
    checked_out_at,
    due_at,
    returned_at,
    notes
)
SELECT
    items.id,
    borrower.id,
    administrator.id,
    administrator.id,
    '2026-07-01 09:00:00+00'::timestamptz,
    '2026-07-08 09:00:00+00'::timestamptz,
    '2026-07-05 15:00:00+00'::timestamptz,
    'Development seed: completed laptop checkout'
FROM items
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'borrower@example.test'
    ORDER BY id
    LIMIT 1
) AS borrower
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'admin@example.test'
    ORDER BY id
    LIMIT 1
) AS administrator
WHERE items.asset_tag = 'EQ-LAP-001'
  AND NOT EXISTS (
      SELECT 1
      FROM checkouts
      WHERE notes = 'Development seed: completed laptop checkout'
  );

INSERT INTO reservations (
    item_id,
    requested_by_user_id,
    starts_at,
    ends_at,
    status,
    notes
)
SELECT
    items.id,
    borrower.id,
    '2030-01-10 09:00:00+00'::timestamptz,
    '2030-01-12 17:00:00+00'::timestamptz,
    'pending',
    'Development seed: pending monitor reservation'
FROM items
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'borrower@example.test'
    ORDER BY id
    LIMIT 1
) AS borrower
WHERE items.asset_tag = 'EQ-MON-001'
  AND NOT EXISTS (
      SELECT 1
      FROM reservations
      WHERE notes = 'Development seed: pending monitor reservation'
  );

INSERT INTO maintenance_records (
    item_id,
    opened_by_user_id,
    assigned_to_user_id,
    status,
    reason,
    description,
    started_at,
    completed_at
)
SELECT
    items.id,
    administrator.id,
    technician.id,
    'completed',
    'Display calibration',
    'Development seed: completed monitor maintenance',
    '2026-06-20 08:00:00+00'::timestamptz,
    '2026-06-20 11:30:00+00'::timestamptz
FROM items
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'admin@example.test'
    ORDER BY id
    LIMIT 1
) AS administrator
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'technician@example.test'
    ORDER BY id
    LIMIT 1
) AS technician
WHERE items.asset_tag = 'EQ-MON-001'
  AND NOT EXISTS (
      SELECT 1
      FROM maintenance_records
      WHERE description = 'Development seed: completed monitor maintenance'
  );

INSERT INTO condition_reports (
    item_id,
    reported_by_user_id,
    checkout_id,
    report_stage,
    condition_rating,
    notes,
    created_at
)
SELECT
    items.id,
    administrator.id,
    checkouts.id,
    'return',
    'good',
    'Development seed: laptop return condition',
    '2026-07-05 15:05:00+00'::timestamptz
FROM checkouts
JOIN items ON items.id = checkouts.item_id
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'admin@example.test'
    ORDER BY id
    LIMIT 1
) AS administrator
WHERE checkouts.notes = 'Development seed: completed laptop checkout'
  AND NOT EXISTS (
      SELECT 1
      FROM condition_reports
      WHERE notes = 'Development seed: laptop return condition'
  );

INSERT INTO condition_reports (
    item_id,
    reported_by_user_id,
    maintenance_record_id,
    report_stage,
    condition_rating,
    notes,
    created_at
)
SELECT
    items.id,
    technician.id,
    maintenance_records.id,
    'maintenance',
    'good',
    'Development seed: monitor post-maintenance condition',
    '2026-06-20 11:35:00+00'::timestamptz
FROM maintenance_records
JOIN items ON items.id = maintenance_records.item_id
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'technician@example.test'
    ORDER BY id
    LIMIT 1
) AS technician
WHERE maintenance_records.description = 'Development seed: completed monitor maintenance'
  AND NOT EXISTS (
      SELECT 1
      FROM condition_reports
      WHERE notes = 'Development seed: monitor post-maintenance condition'
  );

INSERT INTO item_status_history (
    item_id,
    changed_by_user_id,
    previous_status,
    new_status,
    reason,
    source_type,
    source_id,
    changed_at
)
SELECT
    items.id,
    administrator.id,
    'checked_out',
    'available',
    'Development seed: laptop returned',
    'checkout',
    checkouts.id,
    '2026-07-05 15:10:00+00'::timestamptz
FROM checkouts
JOIN items ON items.id = checkouts.item_id
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'admin@example.test'
    ORDER BY id
    LIMIT 1
) AS administrator
WHERE checkouts.notes = 'Development seed: completed laptop checkout'
  AND NOT EXISTS (
      SELECT 1
      FROM item_status_history
      WHERE reason = 'Development seed: laptop returned'
  );

INSERT INTO item_status_history (
    item_id,
    changed_by_user_id,
    previous_status,
    new_status,
    reason,
    source_type,
    source_id,
    changed_at
)
SELECT
    items.id,
    technician.id,
    'maintenance',
    'available',
    'Development seed: monitor maintenance completed',
    'maintenance_record',
    maintenance_records.id,
    '2026-06-20 11:40:00+00'::timestamptz
FROM maintenance_records
JOIN items ON items.id = maintenance_records.item_id
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'technician@example.test'
    ORDER BY id
    LIMIT 1
) AS technician
WHERE maintenance_records.description = 'Development seed: completed monitor maintenance'
  AND NOT EXISTS (
      SELECT 1
      FROM item_status_history
      WHERE reason = 'Development seed: monitor maintenance completed'
  );

INSERT INTO audit_events (
    actor_user_id,
    action,
    entity_type,
    entity_identifier,
    request_id,
    source_ip,
    before_data,
    after_data,
    created_at
)
SELECT
    administrator.id,
    'checkout.returned',
    'checkout',
    checkouts.id::text,
    'dev-seed-checkout-returned',
    '127.0.0.1'::inet,
    '{"status":"checked_out"}'::jsonb,
    '{"status":"available","returned":true}'::jsonb,
    '2026-07-05 15:11:00+00'::timestamptz
FROM checkouts
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'admin@example.test'
    ORDER BY id
    LIMIT 1
) AS administrator
WHERE checkouts.notes = 'Development seed: completed laptop checkout'
  AND NOT EXISTS (
      SELECT 1
      FROM audit_events
      WHERE request_id = 'dev-seed-checkout-returned'
  );

INSERT INTO audit_events (
    actor_user_id,
    action,
    entity_type,
    entity_identifier,
    request_id,
    source_ip,
    before_data,
    after_data,
    created_at
)
SELECT
    technician.id,
    'maintenance.completed',
    'maintenance_record',
    maintenance_records.id::text,
    'dev-seed-maintenance-completed',
    '127.0.0.1'::inet,
    '{"status":"in_progress"}'::jsonb,
    '{"status":"completed"}'::jsonb,
    '2026-06-20 11:41:00+00'::timestamptz
FROM maintenance_records
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE email = 'technician@example.test'
    ORDER BY id
    LIMIT 1
) AS technician
WHERE maintenance_records.description = 'Development seed: completed monitor maintenance'
  AND NOT EXISTS (
      SELECT 1
      FROM audit_events
      WHERE request_id = 'dev-seed-maintenance-completed'
  );

SELECT format(
    'Development seed complete: %s users, %s locations, %s categories, %s items, %s checkouts, %s reservations, %s maintenance records, %s condition reports, %s status history records, %s audit events.',
    (SELECT count(*) FROM users WHERE email IN (
        'admin@example.test',
        'borrower@example.test',
        'audit@example.test',
        'technician@example.test'
    )),
    (SELECT count(*) FROM locations WHERE code IN (
        'HQ',
        'STORAGE-A',
        'OFFICE-101'
    )),
    (SELECT count(*) FROM categories WHERE name IN (
        'Laptop',
        'Monitor',
        'Accessory'
    )),
    (SELECT count(*) FROM items WHERE asset_tag IN (
        'EQ-LAP-001',
        'EQ-MON-001',
        'EQ-ACC-001'
    )),
    (SELECT count(*) FROM checkouts
     WHERE notes = 'Development seed: completed laptop checkout'),
    (SELECT count(*) FROM reservations
     WHERE notes = 'Development seed: pending monitor reservation'),
    (SELECT count(*) FROM maintenance_records
     WHERE description = 'Development seed: completed monitor maintenance'),
    (SELECT count(*) FROM condition_reports WHERE notes IN (
        'Development seed: laptop return condition',
        'Development seed: monitor post-maintenance condition'
    )),
    (SELECT count(*) FROM item_status_history WHERE reason IN (
        'Development seed: laptop returned',
        'Development seed: monitor maintenance completed'
    )),
    (SELECT count(*) FROM audit_events WHERE request_id IN (
        'dev-seed-checkout-returned',
        'dev-seed-maintenance-completed'
    ))
);
