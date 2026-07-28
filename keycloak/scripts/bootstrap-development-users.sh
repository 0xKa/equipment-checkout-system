#!/usr/bin/env bash

set -euo pipefail

: "${KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME:?KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME is required}"
: "${KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD:?KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD is required}"
: "${KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD:?KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD is required}"
: "${KEYCLOAK_SAMPLE_BORROWER_PASSWORD:?KEYCLOAK_SAMPLE_BORROWER_PASSWORD is required}"
: "${KEYCLOAK_AUDIT_VIEWER_PASSWORD:?KEYCLOAK_AUDIT_VIEWER_PASSWORD is required}"

readonly KEYCLOAK_SERVER="http://keycloak:8080"
readonly TARGET_REALM="equipment"
readonly API_CLIENT_ID="equipment-api"
readonly KCADM="/opt/keycloak/bin/kcadm.sh"
readonly KCADM_CONFIG="/tmp/kcadm.config"

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

api_client_json="$(
  "${KCADM}" get clients \
    --config "${KCADM_CONFIG}" \
    --target-realm "${TARGET_REALM}" \
    --query "clientId=${API_CLIENT_ID}" \
    --query "exact=true" \
    --fields id
)"
api_client_uuid="$(printf '%s' "${api_client_json}" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

if [[ -z "${api_client_uuid}" ]]; then
  echo "Required Keycloak client ${API_CLIENT_ID} is missing from realm ${TARGET_REALM}." >&2
  exit 1
fi

for role in employee inventory_admin auditor; do
  if ! "${KCADM}" get "clients/${api_client_uuid}/roles/${role}" \
    --config "${KCADM_CONFIG}" \
    --target-realm "${TARGET_REALM}" >/dev/null; then
    echo "Required Keycloak role ${API_CLIENT_ID}/${role} is missing." >&2
    exit 1
  fi
done

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

echo "Keycloak development-user passwords are configured."
