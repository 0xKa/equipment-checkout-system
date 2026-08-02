#!/usr/bin/env bash

set -euo pipefail

: "${KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME:?KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME is required}"
: "${KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD:?KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD is required}"
: "${KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD:?KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD is required}"
: "${KEYCLOAK_SAMPLE_BORROWER_PASSWORD:?KEYCLOAK_SAMPLE_BORROWER_PASSWORD is required}"
: "${KEYCLOAK_AUDIT_VIEWER_PASSWORD:?KEYCLOAK_AUDIT_VIEWER_PASSWORD is required}"
: "${KEYCLOAK_USER_SYNC_CLIENT_SECRET:?KEYCLOAK_USER_SYNC_CLIENT_SECRET is required}"

readonly KEYCLOAK_SERVER="http://keycloak:8080"
readonly TARGET_REALM="equipment"
readonly USER_SYNC_CLIENT_ID="equipment-user-sync"
readonly LOGIN_THEME="equipment"
readonly KCADM="/opt/keycloak/bin/kcadm.sh"
readonly KCADM_CONFIG="/tmp/kcadm.config"
readonly USER_PROFILE_CONFIG="/opt/keycloak/bootstrap/equipment-user-profile.json"

cleanup() {
  rm -f "${KCADM_CONFIG}"
}
trap cleanup EXIT

ready=false
for ((attempt = 1; attempt <= 60; attempt++)); do
  if {
    printf 'HEAD /health/ready HTTP/1.0\r\n\r\n' >&3
    grep -q 'HTTP/1.0 200' <&3
  } 3<>/dev/tcp/keycloak/9000 2>/dev/null; then
    ready=true
    break
  fi
  sleep 2
done

if [[ "${ready}" != "true" ]]; then
  echo "Keycloak did not become ready for development-user bootstrap." >&2
  exit 1
fi

KC_CLI_PASSWORD="${KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD}" \
  "${KCADM}" config credentials \
  --config "${KCADM_CONFIG}" \
  --server "${KEYCLOAK_SERVER}" \
  --realm master \
  --user "${KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME}" >/dev/null

"${KCADM}" update users/profile \
  --config "${KCADM_CONFIG}" \
  --target-realm "${TARGET_REALM}" \
  --file "${USER_PROFILE_CONFIG}" >/dev/null

"${KCADM}" update "realms/${TARGET_REALM}" \
  --config "${KCADM_CONFIG}" \
  --set "loginTheme=${LOGIN_THEME}" >/dev/null

for role in employee inventory_admin auditor; do
  if ! "${KCADM}" get "roles/${role}" \
    --config "${KCADM_CONFIG}" \
    --target-realm "${TARGET_REALM}" >/dev/null; then
    echo "Required Keycloak realm role ${role} is missing." >&2
    exit 1
  fi
done

user_sync_client_json="$(
  "${KCADM}" get clients \
    --config "${KCADM_CONFIG}" \
    --target-realm "${TARGET_REALM}" \
    --query "clientId=${USER_SYNC_CLIENT_ID}" \
    --query "exact=true" \
    --fields id
)"
user_sync_client_uuid="$(printf '%s' "${user_sync_client_json}" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

if [[ -z "${user_sync_client_uuid}" ]]; then
  echo "Required Keycloak client ${USER_SYNC_CLIENT_ID} is missing from realm ${TARGET_REALM}." >&2
  exit 1
fi

"${KCADM}" update "clients/${user_sync_client_uuid}" \
  --config "${KCADM_CONFIG}" \
  --target-realm "${TARGET_REALM}" \
  --set "secret=${KEYCLOAK_USER_SYNC_CLIENT_SECRET}" >/dev/null

service_account_json="$(
  "${KCADM}" get "clients/${user_sync_client_uuid}/service-account-user" \
    --config "${KCADM_CONFIG}" \
    --target-realm "${TARGET_REALM}" \
    --fields id
)"
service_account_uuid="$(printf '%s' "${service_account_json}" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

if [[ -z "${service_account_uuid}" ]]; then
  echo "Service account for ${USER_SYNC_CLIENT_ID} is missing." >&2
  exit 1
fi

"${KCADM}" add-roles \
  --config "${KCADM_CONFIG}" \
  --target-realm "${TARGET_REALM}" \
  --uid "${service_account_uuid}" \
  --cclientid realm-management \
  --rolename manage-users \
  --rolename view-realm >/dev/null

set_development_password() {
  local username="$1"
  local password="$2"
  local user_json
  local user_uuid

  user_json="$(
    "${KCADM}" get users \
      --config "${KCADM_CONFIG}" \
      --target-realm "${TARGET_REALM}" \
      --query "username=${username}" \
      --query "exact=true" \
      --fields id
  )"
  user_uuid="$(printf '%s' "${user_json}" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

  if [[ -z "${user_uuid}" ]]; then
    echo "Required Keycloak user ${username} is missing from realm ${TARGET_REALM}." >&2
    exit 1
  fi

  KC_CLI_PASSWORD="${password}" \
    "${KCADM}" set-password \
    --config "${KCADM_CONFIG}" \
    --target-realm "${TARGET_REALM}" \
    --userid "${user_uuid}" >/dev/null
}

set_development_password "equipment.admin" "${KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD}"
set_development_password "sample.borrower" "${KEYCLOAK_SAMPLE_BORROWER_PASSWORD}"
set_development_password "audit.viewer" "${KEYCLOAK_AUDIT_VIEWER_PASSWORD}"

echo "Keycloak development service account, theme, and user passwords are configured."
