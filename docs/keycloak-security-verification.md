# Keycloak Security Verification Record

Status date: 2026-07-28

This record summarizes the observed Keycloak integration checks through
Milestone 9. It deliberately distinguishes executed behavior from code
inspection and remaining manual acceptance work.

No bearer token, password, cookie, or client secret was printed, persisted, or
added to the repository during these checks. Automated Go authentication test
files are not authorized at the current checkpoint, so `go test ./...` is not
part of this record.

## Observed results

| Area | Observed result |
| --- | --- |
| Reproducible realm | A fresh disposable Keycloak database imported the canonical realm, fixed users/subjects, clients, scopes, audience, and roles. Repeat bootstrap and normal restart preserved state. |
| Database isolation | A Keycloak-only reset recreated only the Keycloak PostgreSQL volume and preserved an application-database marker. |
| Migration round trip | A uniquely named empty database migrated through version 13, rolled migration 13 down to version 12, and reapplied it to version 13; the temporary database was removed. |
| Seed/JIT ordering | A disposable database containing a JIT-created `equipment.admin` row with null email accepted two seed runs, retained exactly one linked admin row, and produced all four representative local users. |
| API startup gate | Compose waited for application migrations and Keycloak bootstrap. An unavailable JWKS endpoint caused API startup to fail closed. |
| Representative login | Authorization Code with PKCE succeeded for `equipment.admin`, `sample.borrower`, and `audit.viewer`; each resolved to the intended local actor. |
| Public boundary | `/health` and `/ready` remained public and returned `200`; protected routes required Bearer authentication. |
| Header parsing | Missing credentials, malformed schemes, repeated credentials, and `X-Actor-User-ID` without a token returned `401 authentication_required` with `WWW-Authenticate: Bearer`. |
| Invalid token envelope | A structurally invalid compact token returned `401 invalid_token` with the safe invalid-token Bearer challenge. |
| Coarse capabilities | Employee mutation/on-behalf attempts were denied, auditors could read checkout history but not mutate it, and inventory administrators retained inventory/user/all-history access. |
| Ownership | Employee checkout list/get was borrower-filtered in SQL; another borrower's record was hidden with `404`; item-wide history was denied to employees. |
| Trusted attribution | Checkout creator, return recipient, status-history actor, and audit actor consume the verified local actor from request context rather than a client-selected header. |
| JIT concurrency | Eight concurrent first `/api/v1/me` requests for one unseen employee all returned `200`, resolved to one local ID, and created exactly one row. |
| JIT profile policy | Unverified email was omitted; normalized username and initial display name were stored. |
| Identity collision | A Keycloak user matching the existing unlinked `maintenance.tech` username received `403 identity_conflict`; no identity row was created and the local row remained unlinked. |
| Token inspection utility | The allowlisted metadata decoder parsed a synthetic compact JWT without printing token material; PowerShell syntax validation passed. |
| Routine verification | All Go files were formatted; two pinned sqlc generations were clean; Postman JSON and Compose configuration parsed; `go vet ./...` and `go build ./...` passed. |

Temporary Keycloak/application records created for focused checks were removed
by exact identifier after verification.

## Verified by implementation inspection

The current verifier and middleware enforce:

- one compact signed JWT carried by exactly one Bearer credential;
- RS256 only;
- exact configured issuer and `equipment-api` audience;
- required expiry and bounded `nbf`/`iat` clock skew;
- nonblank subject;
- strict `resource_access.equipment-api.roles` shape;
- recognized roles before identity lookup or JIT provisioning;
- exact `(issuer, subject)` local identity resolution;
- cached local JWKS verification rather than per-request Keycloak calls;
- safe public error messages that do not expose verifier or database details.

Inspection is not a substitute for executing every negative cryptographic
case. Those cases remain explicitly listed below.

## Remaining manual acceptance matrix

Use disposable or uniquely named data. Do not reset an established
development database.

- Expired token.
- Future `nbf` and `iat` outside the accepted skew, plus boundary values.
- Wrong issuer and wrong audience.
- Wrong signature, disallowed algorithm, blank subject, and unknown key ID.
- JWKS re-fetch after an actual signing-key rotation.
- Missing or malformed role-claim shapes and a valid token with no recognized
  application role.
- Missing JIT username and verified-email initialization.
- Retention of local JIT profile edits across later logins.
- Immediate denial of a locally deactivated linked account.
- Every cell of the employee/administrator/auditor route matrix.
- Successful employee self checkout/return and administrator on-behalf
  checkout/return with database confirmation of item, checkout, history, and
  audit attribution.
- Rollback confirmation for denied and failed workflow mutations.

## Response expectations

- Missing/malformed credentials: `401 authentication_required`.
- Invalid token: `401 invalid_token`.
- Both `401` cases include an appropriate `WWW-Authenticate: Bearer`
  challenge.
- Valid identity without the required capability: `403 forbidden`.
- Safe JIT profile collision: `403 identity_conflict`.
- Invalid JIT profile: `403 identity_profile_invalid`.
- Locally inactive linked identity: `403 account_inactive`.
- Employee lookup of another user's protected checkout: `404`.
- No failed authentication or authorization path changes workflow, history, or
  audit state.

## Production boundary

The observed Compose environment is a development verification environment,
not a production identity platform. Production TLS/issuer stability, proxy
trust, private administration, secret management, high availability,
backup/restore, upgrades, monitoring, key rotation, revocation expectations,
and abuse protection require separate design and acceptance.
