-- Destructive development-only reset.
-- Preserve goose_db_version so Goose retains the applied migration history.

TRUNCATE TABLE
    audit_events,
    item_status_history,
    condition_reports,
    maintenance_records,
    reservations,
    checkouts,
    items,
    categories,
    locations,
    users
RESTART IDENTITY;
