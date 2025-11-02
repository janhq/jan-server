FROM quay.io/keycloak/keycloak:23.0

ENV KC_HEALTH_ENABLED=true \
    KEYCLOAK_ADMIN=admin \
    KEYCLOAK_ADMIN_PASSWORD=admin \
    KC_HTTP_PORT=8085

COPY import /opt/keycloak/data/import
COPY init /opt/keycloak/init

ENTRYPOINT ["/opt/keycloak/bin/kc.sh", "start", "--import-realm", "--hostname-strict=false", "--http-enabled=true"]
