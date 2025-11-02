#!/usr/bin/env bash
set -euo pipefail

KEYCLOAK_URL=${KEYCLOAK_URL:-http://localhost:8080}
REALM=${KEYCLOAK_REALM:-jan}
ADMIN_USER=${KEYCLOAK_ADMIN:-admin}
ADMIN_PASS=${KEYCLOAK_ADMIN_PASSWORD:-admin}
BACKEND_CLIENT=${BACKEND_CLIENT_ID:-backend}
TARGET_CLIENT=${TARGET_CLIENT_ID:-llm-api}

KCADM="/opt/keycloak/bin/kcadm.sh"

$KCADM config credentials --server "$KEYCLOAK_URL" --realm master --user "$ADMIN_USER" --password "$ADMIN_PASS"

backend_id=$($KCADM get clients -r "$REALM" -q clientId="$BACKEND_CLIENT" --fields id --format csv | tail -n 1 | tr -d '"')
target_id=$($KCADM get clients -r "$REALM" -q clientId="$TARGET_CLIENT" --fields id --format csv | tail -n 1 | tr -d '"')

if [[ -z "$backend_id" || -z "$target_id" ]]; then
  echo "Failed to discover client ids" >&2
  exit 1
fi

$KCADM update clients/$backend_id -r "$REALM" -s "attributes.token-exchange-permissions-enabled=true"
$KCADM update clients/$backend_id -r "$REALM" -s "authorizationServicesEnabled=true"
$KCADM update clients/$target_id -r "$REALM" -s "attributes.token-exchange-permissions-enabled=true"

SERVICE_ACCOUNT="service-account-$BACKEND_CLIENT"

grant_role() {
  local role="$1"
  $KCADM add-roles -r "$REALM" --uusername "$SERVICE_ACCOUNT" --cclientid realm-management --rolename "$role" >/dev/null || true
}

grant_role impersonation
grant_role manage-users
grant_role view-users
grant_role view-realm
grant_role view-clients

$KCADM create clients/$backend_id/authz/resource-server/permission/scope -r "$REALM" -b "{\"name\":\"token-exchange-$TARGET_CLIENT\",\"type\":\"scope\",\"logic\":\"POSITIVE\",\"decisionStrategy\":\"UNANIMOUS\",\"resources\":[\"$target_id\"],\"scopes\":[\"token-exchange\"]}" >/dev/null || true

echo "Token exchange permissions configured for $BACKEND_CLIENT -> $TARGET_CLIENT"
