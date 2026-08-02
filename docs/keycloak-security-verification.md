# Keycloak Security Verification Record

Status date: 2026-08-02

This record separates previously executed authentication checks, current
implementation inspection, and the manual acceptance still required for
synchronous application–Keycloak user management.

No bearer token, password, cookie, or client secret was printed, persisted, or
added to the repository during implementation checks. Automated Go test files
remain outside the authorized scope, so routine verification uses formatting,
sqlc generation, `go vet`, builds, configuration parsing, and focused manual
acceptance.

## Previously observed authentication boundary

| Area | Observed result |
| --- | --- |
| Reproducible realm | A fresh disposable Keycloak database imported the canonical realm, fixed users/subjects, clients, scopes, audience, and roles. Repeat bootstrap and normal restart preserved state. |
| Database isolation | A Keycloak-only reset recreated only the Keycloak PostgreSQL volume and preserved an application-database marker. |
| API startup gate | Compose waited for application migrations and Keycloak bootstrap. An unavailable JWKS endpoint caused API startup to fail closed. |
| Representative login | Authorization Code with PKCE succeeded for `equipment.admin`, `sample.borrower`, and `audit.viewer`; each resolved to the intended local actor. |
| Public boundary | `/health` and `/ready` remained public and returned `200`; protected routes required Bearer authentication. |
| Header parsing | Missing credentials, malformed schemes, repeated credentials, and `X-Actor-User-ID` without a token returned `401 authentication_required` with `WWW-Authenticate: Bearer`. |
| Invalid token envelope | A structurally invalid compact token returned `401 invalid_token` with the safe invalid-token Bearer challenge. |
| Coarse capabilities | Employee mutation/on-behalf attempts were denied, auditors could read checkout history but not mutate it, and inventory administrators retained inventory/user/all-history access. |
| Ownership | Employee checkout list/get was borrower-filtered in SQL; another borrower's record was hidden with `404`; item-wide history was denied to employees. |
| Trusted attribution | Checkout creator, return recipient, status-history actor, and audit actor consume the verified local actor from request context rather than a client-selected header. |
| Token inspection utility | The allowlisted metadata decoder parsed a synthetic compact JWT without printing token material. |

The former JIT-provisioning observations are historical and no longer describe
the runtime. Authentication now performs exact linked identity lookup only.

## Current implementation-backed controls

- The application API is the supported administrative writer for managed users.
- Keycloak and application PostgreSQL remain isolated; the application uses the
  Admin REST API and never writes Keycloak tables directly.
- GoCloak is confined to one adapter behind project-owned domain types.
- Every complete-state admin operation obtains one fresh bounded
  client-credentials token and reuses it for its Keycloak calls.
- The `equipment-user-sync` service account has `manage-users` and
  `view-realm`; it is not assigned `realm-admin`.
- Bootstrap installs a managed, admin-only `equipment_display_name` user-profile
  attribute, so the application never invents first or last names.
- User creation sends the complete profile, role, and activation state through
  one provider operation and compensates partial provider or local-row failure
  by deleting the new identity.
- Profile, role, and activation changes share one service-owned workflow: lock
  and snapshot the local row, replace complete Keycloak state, write the same
  PostgreSQL state, commit, and attempt one snapshot restoration on any failure
  after Keycloak may have changed.
- Soft deprovisioning disables both sides and preserves the local row, checkout
  history, status history, and audit references.
- Temporary passwords are passed directly to Keycloak as temporary credentials;
  they are not stored, returned, trimmed, or intentionally logged.
- Authentication requires exactly one recognized application realm role and an
  exact active local `(issuer, subject)` link. Unknown identities receive
  `403 identity_not_linked` and cannot create local rows.
- The one-shot reconciler reuses the complete-state provider operation, pushes
  local intended state, provisions unlinked local rows, refuses conflict
  auto-linking, and reports rather than deletes orphans.
- Public errors map invalid roles, unlinked identities, provider conflicts,
  missing linked identities, and provider unavailability without exposing
  GoCloak responses or credentials.

## Manual acceptance matrix

Use disposable or uniquely named data. Do not reset an established development
database.

- Apply migration 14 and verify the four seed roles plus the named role check.
- Reset a disposable Keycloak realm and verify the service-account secret and
  limited realm-management roles.
- Create a user and confirm one linked local row, the exact Keycloak username,
  email, `equipment_display_name`, enabled state, and one realm role.
- Exercise local and Keycloak username/email conflicts and confirm no duplicate
  or partially created managed user remains.
- Force a failure after Keycloak creation and confirm compensating deletion.
- Update profile, role, activation, and soft-deprovision state and compare both
  systems.
- Confirm checkout, status-history, and audit foreign keys remain valid after
  deprovisioning.
- Obtain a fresh token after a role change and verify the selected capability
  matrix. Confirm an old token can retain its role only through the five-minute
  token lifetime.
- Deactivate a user and confirm an already issued token is immediately denied by
  the local inactive-account check.
- Set a temporary password and verify replacement is required at next sign-in;
  inspect application rows, responses, and logs for absence of the password.
- Create an unmanaged Keycloak user and confirm `/api/v1/me` returns
  `403 identity_not_linked` without inserting a local row.
- Run `make reconcile-users`; confirm it provisions `maintenance.tech`, repairs
  deliberate linked drift, refuses a username/email conflict, reports a
  Keycloak-only orphan, and does not delete the orphan.
- Repeat the prior cryptographic cases: expiry, future `nbf`/`iat`, wrong issuer,
  wrong audience, wrong signature, disallowed algorithm, blank subject, unknown
  key ID, and signing-key rotation.
- Verify missing/malformed role claims, no recognized role, and multiple
  recognized roles are rejected.

## Response expectations

- Missing or malformed credentials: `401 authentication_required`.
- Invalid token: `401 invalid_token`.
- Both `401` cases include the appropriate `WWW-Authenticate: Bearer` challenge.
- Unknown exact identity: `403 identity_not_linked`.
- Locally inactive linked identity: `403 account_inactive`.
- Valid identity without a required capability: `403 forbidden`.
- Invalid managed role: `400 invalid_user_role`.
- Local uniqueness conflict: `409 username_conflict` or `409 email_conflict`.
- Keycloak uniqueness conflict: `409 keycloak_conflict`.
- Keycloak user-profile or password-policy rejection: `400 keycloak_rejected`.
- Unlinked local managed mutation: `409 keycloak_identity_unlinked`.
- Missing linked Keycloak identity: `409 keycloak_identity_not_found`.
- Bounded Keycloak administration failure: `503 service_unavailable`.
- Employee lookup of another user's protected checkout: `404`.

## Production boundary

The Compose environment is a development verification environment, not a
production identity platform. Production TLS and issuer stability, proxy trust,
secret storage and rotation, private administration, high availability,
backup/restore, monitoring, Keycloak upgrades, signing-key rotation, revocation,
and abuse protection require separate design and acceptance. Synchronous
compensation reduces ordinary partial failures but cannot provide a distributed
transaction across PostgreSQL and Keycloak; the one-shot reconciler is the
explicit recovery path.
