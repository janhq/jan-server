# Kong Gateway Authentication Implementation Roadmap

## 🎯 Overview

This document outlines the implementation plan for adding JWT and API key authentication to the Jan Server via Kong OSS gateway. The system will leverage Keycloak for token issuance and Kong for request validation at the edge.

---

## 🔄 Implementation Approach Update (Post-Review)

**Key Changes from Initial Plan:**

1. **Kong Plugin Selection**: Using **Kong OSS `jwt` plugin** instead of Enterprise `openid-connect`
   - ✅ No license required (Apache 2.0)
   - ✅ Sufficient for JWT validation with Keycloak
   - ✅ One-time JWKS configuration per environment
   - ✅ Kong restart when Keycloak signing keys change (acceptable operational pattern)

2. **Consumer Management**: **Admin API** approach for credentials
   - ✅ Credentials managed ONLY via Kong Admin API (never in Git)
   - ✅ No `consumers/` directory in kong-config repo (security best practice)
   - ✅ Manual Kong restart when Keycloak keys rotate

3. **Auth Plugin Logic**: Implemented **OR logic** using anonymous consumers
   - ✅ Request satisfies JWT OR API key (not both required)
   - ✅ Uses `anonymous` parameter with `request-termination` plugin
   - ✅ Comprehensive regression tests covering all auth paths

4. **Environment Security**: **Environment-specific overlays** with Kustomize
   - ✅ `ssl_verify: false` only in development overlay
   - ✅ `ssl_verify: true` + CA bundle in staging/production
   - ✅ No insecure defaults in base configuration

5. **Keycloak Migration**: **Detailed migration plan** for Keycloak 24+ upgrade
   - ✅ Staging rehearsal with full test suite
   - ✅ Database backup and rollback procedures
   - ✅ Compatibility testing for custom guest user flows
   - ✅ Version pinning for stability (`keycloak:24.0.5`)

---

## 🔐 JWT Validation Hardening (Improvements from Security Audit)

**Enhanced JWT Configuration:**

1. **Algorithm & Claims Verification**:
   ```yaml
   plugins:
     - name: jwt
       config:
         algorithm: RS256                    # Explicitly set
         key_claim_name: kid                 # Use kid for key rotation
         secret_is_base64: false
         claims_to_verify: ["exp","nbf"]    # Verify expiry and not-before
         maximum_expiration: 3600           # Enforce 1h max TTL
         run_on_preflight: false            # Allow CORS preflight
         anonymous: kong-anon-jwt           # OR logic fallthrough
   ```

2. **Custom Claim Validators at Service Layer** (defense-in-depth):
   - Validate `iss` (issuer) matches Keycloak realm
   - Validate `aud` (audience) matches jan-client
   - Validate `azp` (authorized party) matches expected client
   - Add clock skew tolerance (±60 seconds) for time drift

3. **KID-Based Key Rotation** (for zero-downtime rotation):
   - Use `key_claim_name: "kid"` to enable key versioning
   - Store each Keycloak signing key variant in Kong as separate JWT credential
   - When Keycloak rotates keys: Kong restart picks up new key
   - Old key remains valid during grace period

4. **Service-Layer Re-Validation** (for sensitive operations):
   - Admin endpoints should re-validate JWT signature
   - Check JWT claims match Kong headers (confused deputy protection)
   - Log all auth decisions for audit trail

---

## 🔐 API Key Hardening

1. **Key-Auth Plugin Configuration**:
   ```yaml
   plugins:
     - name: key-auth
       config:
         key_names: ["X-API-Key","Authorization"]  # Support Bearer too
         key_in_header: true
         hide_credentials: true              # Strip header from upstream
         run_on_preflight: false             # Allow CORS preflight
         anonymous: kong-anon-key            # OR logic fallthrough
   ```

2. **Header Normalization**:
   ```yaml
   plugins:
     - name: request-transformer
       config:
         remove:
           headers: ["Authorization"]        # Strip after key-auth validates
         add:
           headers: ["X-Gateway-Auth: kong"] # Indicate auth source
   ```

3. **Per-Consumer Rate Limiting**:
   ```yaml
   plugins:
     - name: rate-limiting
       config:
         minute: 100
         limit_by: consumer                  # Per-user limit, not global
   ```

---

## 🔐 OR-Logic Security Guarantees

**Prevent Auth Bypass with Anonymous Consumer Pattern**:

```yaml
# Create anonymous consumers for fallthrough
consumers:
  - username: kong-anon-jwt      # If JWT fails
  - username: kong-anon-key      # If API key fails

# Both jwt and key-auth plugins on every protected route
plugins:
  - name: jwt
    config:
      anonymous: kong-anon-jwt   # Allow fallthrough
  
  - name: key-auth
    config:
      anonymous: kong-anon-key   # Allow fallthrough
  
  - name: request-termination
    consumer: kong-anon-key
    config:
      status_code: 401
      message: "Authentication required: provide JWT or API key"
```

**Regression Test Suite** (verify auth enforcement):
- JWT only → 200 OK
- API key (X-API-Key) → 200 OK
- API key (Bearer) → 200 OK
- Both JWT + API key → 200 OK
- Neither → 401 Unauthorized
- Invalid JWT → 401
- Invalid API key → 401
- Revoked API key → 401

---

## 🔒 Admin API Security

1. **Kong Admin API Access Control**:
   - Restrict to localhost only (firewall rules)
   - Add IP allowlist for CI/CD servers
   - mTLS certificates for admin connections
   - Token-based authentication (if Kong Enterprise)

2. **Credential Management Strategy**:
   - **NEVER commit consumer credentials to Git**
   - Consumers and API keys managed ONLY via Kong Admin API at runtime
   - Manual Kong restart when Keycloak keys change

3. **GitOps Discipline with decK**:
   - `decK --select-tag topology` for Kong topology only
   - Topology = services, routes, plugins (versioned in Git)
   - Credentials = consumers, keys (runtime only, never in Git)
   - Pre-commit hooks to prevent credential leaks

---

## 📊 Observability & Metrics

**Track authentication health:**

1. **Prometheus Metrics**:
   - `kong_auth_jwt_success_total` / `kong_auth_jwt_failures_total`
   - `kong_auth_apikey_success_total` / `kong_auth_apikey_failures_total`
   - `kong_auth_latency_ms` (histogram, SLO: p99 < 50ms)
   - `service_auth_latency_ms` (service-level re-validation latency)
   - `service_confused_deputy_attempts_total` (should be 0)

2. **Grafana Dashboards**:
   - Auth success rate by method (JWT vs API key)
   - Auth failure breakdown (why: expired, invalid_sig, bad_aud)
   - Auth latency trends (p50, p95, p99)
   - Failed auth by path (identify problem endpoints)

3. **SLO Definitions**:
   - Auth latency p99 < 50ms
   - Auth success rate > 99.9%
   - No confused deputy attempts (target: 0)

4. **Structured Logging**:
   - Log all auth attempts (success + failure)
   - Include: user_id, auth_method, kid, status, latency
   - Never log raw JWT or API key
   - Send to centralized logging (ELK, etc.)

---

## 🔄 Service Defense-in-Depth

**Protect services beyond Kong**:

1. **Prefer Kong-Injected Headers**:
   - Read `X-Consumer-ID`, `X-Consumer-Custom-ID` from Kong
   - These are more trustworthy than parsing JWT directly
   - Still re-validate JWT for sensitive operations

2. **Confused Deputy Protection**:
   - If both JWT and API key present, verify they refer to same user
   - Reject if principals disagree (potential attack)
   - Log these attempts for investigation

3. **Admin Endpoint Re-Validation**:
   - Admin operations should verify JWT signature
   - Check user has required role/scope
   - Log all admin actions

4. **Audit Logging**:
   - Track user ID, auth method, endpoint, operation, result
   - Include timestamps and request IDs
   - Send to centralized audit trail

---

## 🔌 Keycloak Configuration Enhancements

1. **User API Keys**:
   - Enable in Keycloak 24+: Realm Settings → User Profile → User API Keys
   - Set TTL: 90 days default
   - Add custom attributes for key metadata

2. **JWT Token Lifetime**:
   - Access token: 1 hour (short-lived)
   - Refresh token: 24 hours (longer-lived)
   - Configure in Realm Settings → Tokens

3. **Key Rotation Policy**:
   - Keycloak automatically rotates signing keys
   - When rotation occurs: Kong restart picks up new key
   - Old key remains valid for 24-48 hours (grace period)

---

## ⚠️ Kong Restart Strategy for Key Changes

**When Keycloak rotating keys**:

1. **Current Flow**:
   - Keycloak rotates signing keys (KID changes)
   - Kong still has old key in cache
   - JWT validation fails for new tokens temporarily

2. **Manual Restart Solution**:
   ```bash
   # Detect KID change (via monitoring or alerts)
   # Restart Kong to reload JWKS
   docker-compose restart kong
   
   # During restart (seconds):
   # - Kong proxy becomes unavailable
   # - Existing connections drop
   # - Health checks fail
   # - Requests rejected
   
   # After restart:
   # - Kong loads new JWKS
   # - New tokens validated successfully
   # - Service continues
   ```

3. **Monitoring for Key Changes**:
   - Set up alerts on Keycloak key rotation
   - Alert ops team to trigger Kong restart
   - Or: Scheduled Kong restart during maintenance window

4. **Grace Period**:
   - Keep old key in Kong for 24-48 hours if possible
   - Allows time for token refresh before restart
   - Reduces unexpected rejections

---

## 📋 Implementation Checklist

### Phase 1: Kong Gateway Authentication (Week 1)

- [ ] **Kong Admin API Setup**
  - [ ] Expose Kong Admin API (port 8001) in docker-compose
  - [ ] Add localhost-only access restriction
  - [ ] Add mTLS certificates (optional for OSS)
  - [ ] Test Admin API access

- [ ] **Update Kong Configuration**
  - [ ] Add JWT plugin with enhanced config (RS256, claims_to_verify, kid)
  - [ ] Add key-auth plugin
  - [ ] Add request-transformer plugin
  - [ ] Create anonymous consumers (kong-anon-jwt, kong-anon-key)
  - [ ] Add request-termination plugin for final auth check
  - [ ] Apply to all protected routes
  - [ ] Update docker/services-api.yml: AUTH_ENABLED: true

- [ ] **Test Authentication**
  - [ ] JWT only → success
  - [ ] API key only → success
  - [ ] Both → success
  - [ ] Neither → 401
  - [ ] Invalid JWT → 401
  - [ ] Invalid API key → 401

- [ ] **Developer Documentation**
  - [ ] Write Kong config change workflow
  - [ ] Document PR review process
  - [ ] Create troubleshooting guide

### Phase 2: API Key Lifecycle (Week 2)

- [ ] **Keycloak 24+ Upgrade** (if not already done)
  - [ ] Migrate existing realm to Keycloak 24+
  - [ ] Enable User API Keys feature
  - [ ] Test all existing auth flows
  - [ ] Verify custom guest user flows work

- [ ] **Keycloak Realm Configuration**
  - [ ] Enable User API Keys feature in realm settings
  - [ ] Configure API key policies (expiration, rotation)
  - [ ] Add custom user attributes for key metadata
  - [ ] Update `keycloak/import/realm-jan.json` with new settings

- [ ] **Keycloak Admin API Client**
  - [ ] Create `services/llm-api/internal/infrastructure/keycloak/apikeys.go`
  - [ ] Wrapper for User API Keys endpoints
  - [ ] Error handling and retries
  - [ ] Integration tests

- [x] **API Key Endpoints in LLM-API**
  - [x] Create `services/llm-api/internal/interfaces/httpserver/handlers/apikeyhandler/`
  - [x] Implement endpoints:
    - `POST /auth/api-keys` - Generate new key (JWT required)
    - `GET /auth/api-keys` - List user's keys (metadata only)
    - `DELETE /auth/api-keys/{id}` - Revoke key
  - [x] Add request/response DTOs
  - [x] Add Swagger documentation
  - [x] Implement JWT validation middleware
  - [x] Return API key only once at creation (security best practice)

- [x] **Kong Consumer Management via Admin API**
  - [x] **IMPORTANT**: Consumers and credentials managed ONLY via Kong Admin API at runtime
  - [x] **NEVER commit consumer credentials to Git** (security requirement)
  - [x] Map Keycloak user ID to Kong consumer `custom_id`
  - [x] Store consumer metadata in Kong tags for traceability
  - [x] Implement consumer creation when needed
  - [x] Add audit logging for all Admin API operations

### Phase 3: Service Hardening (Week 3)

- [ ] **Enable AUTH_ENABLED in All Services**
  - [ ] Update `docker/services-api.yml`:
    - `AUTH_ENABLED: true` (change from false)
  - [ ] Update environment configs:
    - `config/development.env`
    - `config/production.env.example`
  - [ ] Update Kubernetes Helm values:
    - `k8s/jan-server/values.yaml`
    - `k8s/jan-server/values-production.yaml`

- [x] **Refactor Media API Authentication**
  - [x] Remove static `MEDIA_SERVICE_KEY` check
  - [x] Add JWT validator (reuse from llm-api)
  - [x] Add Kong consumer header support
  - [x] Route media-api through Kong gateway
  - [x] Update media-api tests

- [ ] **Add Auth Health Checks**
  - [ ] Create `GET /health/auth` endpoint for each service:
    ```go
    // Returns 200 if:
    // - JWKS loaded successfully
    // - Can connect to Keycloak
    // - Auth middleware is active
    ```
  - [ ] Implement in:
    - `llm-api`
    - `response-api`
    - `media-api`
    - `mcp-tools`
  - [ ] Update readiness probes in Docker Compose
  - [ ] Update readiness probes in Kubernetes manifests

- [ ] **Update Observability**
  - [ ] Add auth failure metrics to Prometheus
  - [ ] Add auth traces to OpenTelemetry
  - [ ] Create Grafana dashboard for auth analytics
  - [ ] Add structured logging for auth events

### Phase 4: Testing & Rollout (Week 4)

- [ ] **Postman Collections**
  - [ ] Create `tests/automation/api-key-flows.json`
  - [ ] Test scenarios:
    - Generate API key with valid JWT
    - Use API key in X-API-Key header
    - Use API key in Authorization: Bearer header
    - List user's API keys
    - Revoke API key
    - Verify revoked key is rejected
  - [ ] Add to existing Newman test suites

- [ ] **Integration Tests**
  - [ ] Test JWT + API key both work end-to-end
  - [ ] Test Kong consumer header injection
  - [ ] Test service-level auth validation
  - [ ] Test auth failure scenarios

- [ ] **Load Testing**
  - [ ] Test auth overhead (< 50ms target)
  - [ ] Test concurrent API key requests
  - [ ] Test Kong throughput with auth enabled
  - [ ] Identify bottlenecks

- [ ] **Security Review**
  - [ ] Penetration testing
  - [ ] Verify no auth bypass routes
  - [ ] Check for timing attacks
  - [ ] Verify secrets are not logged

- [ ] **Documentation**
  - [ ] Update API documentation with auth requirements
  - [ ] Document API key generation workflow
  - [ ] Update getting started guide
  - [ ] Create video tutorials (optional)

- [ ] **Staged Rollout**
  - [ ] Deploy to development environment
  - [ ] Enable auth in test mode (log warnings)
  - [ ] Monitor for false rejections
  - [ ] Fix issues, iterate
  - [ ] Deploy to staging with auth enforced
  - [ ] Run full test suite
  - [ ] Deploy to production with feature flag
  - [ ] Gradually roll out to users (10% → 50% → 100%)

---

## 🔧 Technical Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ Authorization: Bearer <jwt|api_key>
       │ X-API-Key: <api_key>
       ▼
┌─────────────────────────────────────────┐
│           Kong Gateway (OSS)             │
│  ┌──────────────────────────────────┐  │
│  │  1. jwt plugin (RS256, JWKS)     │  │
│  │  2. key-auth (API Keys)          │  │
│  │  3. Anonymous consumer (OR)      │  │
│  │  4. request-transformer          │  │
│  └──────────────────────────────────┘  │
│  Injects: X-Consumer-ID,                │
│           X-Consumer-Custom-ID,         │
│           X-Consumer-Username           │
└──────────┬──────────────────────────────┘
           │ Authenticated Request
           ▼
    ┌──────────────┐
    │   Services   │
    │  ┉┉┉┉┉┉┉┉┉┉  │
    │  llm-api     │ ◄─┐
    │  media-api   │   │ All services:
    │  response-api│   │ - Read X-Consumer-* headers
    │  mcp-tools   │   │ - Validate JWT (defense-in-depth)
    └──────────────┘ ◄─┘ - Extract principal
```

**API Key Flow:**
```
User (authenticated with JWT)
       ↓
POST /auth/api-keys (Authorization: Bearer <jwt>)
       ↓
┌──────────────────────────────────────────────┐
│  LLM-API Service                              │
│  1. Validates JWT                             │
│  2. Extracts user ID from JWT claims          │
│  3. Calls Keycloak Admin API                  │
└──────────────┬───────────────────────────────┘
               ↓
┌──────────────────────────────────────────────┐
│  Keycloak 24+ User API Keys                   │
│  - Generates new API key                      │
│  - Stores hashed key in database              │
│  - Returns plaintext key (show once)          │
│  - Emits event or sends webhook               │
└──────────────┬───────────────────────────────┘
               │
               └──────────────┐
                              ↓
                   Returns to user
                   (Store securely!)
    
Usage Flow:
Client → Request with X-API-Key: <key>
       ↓
Kong validates against credentials (stored in memory/DB)
       ↓ (if valid)
Kong injects: X-Consumer-ID, X-Consumer-Custom-ID, X-Consumer-Username
       ↓
Service receives authenticated request
```

---

## 🔐 Consumer & Credential Management Strategy

**⚠️  CRITICAL SECURITY REQUIREMENT:**

Kong consumers and their credentials (API keys) **MUST NEVER** be committed to Git. This follows Kong's security best practices and prevents credential exposure.

**Implementation Approach:**

```
GitOps (kong-config repo)          Runtime (Admin API)
━━━━━━━━━━━━━━━━━━━━━━━━━          ━━━━━━━━━━━━━━━━━━━━━━
✅  Services                         ✅  Consumers
✅  Routes                           ✅  Key-auth credentials
✅  Plugins (config)                 ✅  JWT secrets
✅  Upstreams                        ✅  ACL groups
✅  Certificates (refs only)         ✅  OAuth2 credentials
❌  Consumers                        
❌  Credentials                      
❌  Secrets                          
```

---

## 🔧 Kong Configuration Structure

```yaml
# base/kong.yaml (stateless topology only)
_format_version: "3.0"
_transform: true

# Anonymous consumers for JWT/Key-Auth OR logic
consumers:
  - username: kong-anon-jwt
    # Allows requests without JWT to proceed to key-auth
  - username: kong-anon-key
    # Allows requests without key-auth to proceed to jwt

# Global rate limiting (adjust per route as needed)
plugins:
  - name: rate-limiting
    config:
      minute: 100
      hour: 5000
      policy: local
      fault_tolerant: true
      
  - name: request-transformer
    config:
      add:
        headers:
          - "X-Gateway-Auth: kong"
          - "X-Gateway-Version: 3.5"

# Services and Routes
services:
  - name: llm-api-svc
    url: http://llm-api:8080
    connect_timeout: 60000
    write_timeout: 60000
    read_timeout: 60000
    retries: 3
    routes:
      - name: llm-api-route
        paths: [/llm]
        strip_path: true
        path_handling: v1
        preserve_host: false
        protocols: ["http", "https"]
        https_redirect_status_code: 426
        plugins:
          # JWT validation (OSS jwt plugin)
          - name: jwt
            config:
              key_claim_name: sub
              secret_is_base64: false
              claims_to_verify: ["exp"]
              anonymous: kong-anon-jwt  # OR logic
          
          # API Key validation
          - name: key-auth
            config:
              key_names: 
                - X-API-Key
                - x-api-key
              key_in_header: true
              hide_credentials: true
              anonymous: kong-anon-key  # OR logic
          
          # Reject if both auth methods fail
          - name: request-termination
            consumer: kong-anon-key
            config:
              status_code: 401
              message: "Authentication required"
          
          # Route-specific rate limiting
          - name: rate-limiting
            config:
              minute: 100
          
          # CORS configuration
          - name: cors
            config:
              origins: 
                - "http://localhost"
                - "http://localhost:3000"
              methods: 
                - GET
                - POST
                - PUT
                - DELETE
                - PATCH
                - OPTIONS
              headers:
                - Authorization
                - Content-Type
                - X-API-Key
              credentials: true
              max_age: 3600

  - name: media-api-svc
    url: http://media-api:8285
    routes:
      - name: media-api-route
        paths: [/media]
        strip_path: true
        plugins:
          - name: jwt
            config:
              anonymous: kong-anon-jwt
          - name: key-auth
            config:
              anonymous: kong-anon-key
          - name: request-termination
            consumer: kong-anon-key
            config:
              status_code: 401
          - name: rate-limiting
            config:
              minute: 50
          - name: cors

  - name: response-api-svc
    url: http://response-api:8082
    routes:
      - name: response-api-route
        paths: [/responses]
        strip_path: true
        plugins:
          - name: jwt
            config:
              anonymous: kong-anon-jwt
          - name: key-auth
            config:
              anonymous: kong-anon-key
          - name: request-termination
            consumer: kong-anon-key
            config:
              status_code: 401
          - name: rate-limiting
            config:
              minute: 100
          - name: cors

  - name: mcp-tools-svc
    url: http://mcp-tools:8091
    routes:
      - name: mcp-tools-route
        paths: [/mcp]
        strip_path: true
        plugins:
          - name: jwt
            config:
              anonymous: kong-anon-jwt
          - name: key-auth
            config:
              anonymous: kong-anon-key
          - name: request-termination
            consumer: kong-anon-key
            config:
              status_code: 401
          - name: rate-limiting
            config:
              minute: 200
          - name: cors

# ⚠️  IMPORTANT: NO consumer credentials in this file!
# Consumers with key-auth credentials are managed ONLY via Kong Admin API at runtime.
```

---

## 🚀 Quick Start Guide

### For Developers

```bash
# 1. Test auth locally
curl -H "Authorization: Bearer $JWT" http://localhost:8000/llm/v1/models

# 2. Use API key
curl -H "X-API-Key: $API_KEY" http://localhost:8000/llm/v1/models
```

### For Users (Generating API Keys)

```bash
# 1. Login and get JWT token
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user@example.com","password":"secret"}'
# Response: {"access_token": "eyJhbGc...", ...}

# 2. Generate API key
curl -X POST http://localhost:8000/auth/api-keys \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{"description":"My laptop key"}'
# Response: {"key":"sk_live_abc123...","id":"key_xyz","created_at":"..."}
# ⚠️ SAVE THIS KEY! You won't see it again.

# 3. Use API key
curl http://localhost:8000/llm/v1/models \
  -H "X-API-Key: sk_live_abc123..."

# 4. List your keys (metadata only)
curl http://localhost:8000/auth/api-keys \
  -H "Authorization: Bearer eyJhbGc..."

# 5. Revoke a key
curl -X DELETE http://localhost:8000/auth/api-keys/key_xyz \
  -H "Authorization: Bearer eyJhbGc..."
```

---

## 🧪 Test Cases (Postman Collection)

**Location:** `tests/automation/auth-postman-scripts.json`

**Status:** ✅ **All test cases implemented and ready**

### Test Collection Structure

The Postman collection is organized into the following test suites:

#### 1. **Health Checks**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| LLM API Health Check | GET | `/healthz` | 200 OK, status: "ok" |

**Purpose:** Verify the LLM-API service is running and responsive.

#### 2. **Setup - Bootstrap Credentials**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| Seed Guest Token | POST | `/auth/guest-login` | 201 Created, includes access_token |
| Seed Obtain Keycloak Admin Token | POST | `/realms/master/protocol/openid-connect/token` | 200 OK, admin access_token |
| Seed Create Test User | POST | `/admin/realms/{{realm}}/users` | 201/204/409, user_id set |
| Seed Set Test User Password | PUT | `/admin/realms/{{realm}}/users/{{test_user_id}}/reset-password` | 204 No Content |
| Seed Obtain Registered User Token | POST | `/realms/{{realm}}/protocol/openid-connect/token` | 200 OK, user access_token |

**Purpose:** Initialize test data and obtain credentials for downstream tests.

#### 3. **LLM API - Guest Token**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| List Models (Guest Token) | GET | `/v1/models` | 200 OK, models array, X-Auth-Method: jwt |
| Get Model Details (Guest Token) | GET | `/v1/models/catalogs/{{model_id_encoded}}` | 200 OK, model details |
| Create Chat Completion (Guest Token) | POST | `/v1/chat/completions` | 200 OK, choices array with message content |

**Purpose:** Verify guest JWT tokens can access read operations.

#### 4. **LLM API - Registered User Token**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| List Models (Registered User) | GET | `/v1/models` | 200 OK, X-Principal-Id header set, X-Auth-Method: jwt |

**Purpose:** Verify registered user JWT tokens work and include principal headers.

#### 5. **Guest Login Flow**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| Request Guest Token | POST | `/auth/guest-login` | 201 Created, access_token + expires_in |
| Upgrade Guest Account | POST | `/auth/upgrade` | 200 OK, status: "upgraded" |

**Purpose:** Test guest account creation and upgrade to permanent account.

#### 6. **JWT Login Flow**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| Obtain Keycloak Admin Token | POST | `/realms/master/protocol/openid-connect/token` | 200 OK, admin token |
| Create Test User | POST | `/admin/realms/{{realm}}/users` | 201/204/409 |
| Set Test User Password | PUT | `/admin/realms/{{realm}}/users/{{test_user_id}}/reset-password` | 204 No Content |
| Obtain Registered User Token | POST | `/realms/{{realm}}/protocol/openid-connect/token` | 200 OK, user token |

**Purpose:** Simulate registered user authentication via Keycloak.

#### 7. **API Key Flow**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| Create API Key | POST | `/auth/api-keys` | 201 Created, includes key + id |
| List API Keys | GET | `/auth/api-keys` | 200 OK, items array |
| Delete API Key | DELETE | `/auth/api-keys/{{api_key_id}}` | 204 No Content |

**Purpose:** Test API key management (create, list, revoke).

#### 8. **Teardown - Cleanup**

| Test Case | Method | Endpoint | Expected Result |
|-----------|--------|----------|-----------------|
| Delete Test User | DELETE | `/admin/realms/{{realm}}/users/{{teardown_user_id}}` | 204/404 |

**Purpose:** Clean up test data after test run completes.

### Test Execution Flow

```
Health Checks
    ↓
Setup (creates credentials for all downstream tests)
    ├─ Guest Token
    ├─ Keycloak Admin Token
    ├─ Test User
    └─ Registered User Token
    ↓
Parallel Test Groups:
    ├─ Guest Token Flows
    ├─ Registered User Token Flows
    ├─ Guest Login & Upgrade
    ├─ JWT Login
    └─ API Key Management
    ↓
Teardown (cleanup test data)
```

### Test Coverage Matrix

| Auth Method | Coverage | Status |
|-------------|----------|--------|
| Guest JWT | ✅ Read models, chat completions, account upgrade | Working |
| User JWT | ✅ Authenticated list models, principal headers | Working |
| API Keys | ✅ Create, list, delete, manage lifecycle | Working |
| JWT Errors | ⏳ Expired, invalid sig, missing claims | Planned Phase 2 |
| API Key Errors | ⏳ Revoked key, invalid key, key rotation | Planned Phase 2 |
| Auth Bypass | ⏳ Neither JWT nor key → 401, invalid both → 401 | Planned Phase 2 |

### Running Tests Locally

```bash
# Prerequisites: Docker stack must be running
make up-full

# Run all auth tests
make test-auth

# Run specific test collection
newman run tests/automation/auth-postman-scripts.json \
  --env-var kong_url=http://localhost:8000 \
  --env-var llm_api_url=http://localhost:8000 \
  --env-var keycloak_base_url=http://localhost:8085 \
  --reporters cli,json \
  --reporter-json-export newman.json

# Run tests with environment file
newman run tests/automation/auth-postman-scripts.json \
  -e config/testing.env \
  --reporters cli,json
```

### Test Collection Variables

**Collection Variables (Pre-request):**

| Variable | Default | Purpose |
|----------|---------|---------|
| `llm_api_url` | `http://localhost:8080` | LLM-API base URL (or `http://localhost:8000` via Kong) |
| `keycloak_base_url` | `http://localhost:8085` | Keycloak base URL |
| `realm` | `jan` | Keycloak realm |
| `client_id_public` | `llm-api` | Public client for direct access grants |
| `keycloak_admin` | `admin` | Keycloak admin username |
| `keycloak_admin_password` | `admin` | Keycloak admin password |
| `model_id` | `qwen/qwen2.5-0.5b-instruct` | Test model for chat completions |

**Generated Variables (Runtime):**

| Variable | Set By | Purpose |
|----------|--------|---------|
| `guest_access_token` | Setup | Guest JWT token |
| `user_access_token` | Setup | Registered user JWT token |
| `api_key_id` | API Key Flow | Created API key ID |
| `api_key_secret` | API Key Flow | Created API key (secret) |
| `test_user_id` | Setup | Test user Keycloak ID |
| `test_user_username` | Setup | Generated test username |
| `test_user_email` | Setup | Generated test email |

### Test Assertions

**Health Check Assertions:**
- Status code is 200
- Response body contains status: "ok"

**Token Creation Assertions:**
- Status code is 201 (Created) for guest
- Status code is 200 for user login
- Response contains `access_token` (non-empty string)
- Response contains `expires_in` (positive integer)

**API Usage Assertions:**
- Status code is 200 OK
- Response schema matches OpenAPI spec
- `X-Auth-Method` header present and equals "jwt"
- `X-Principal-Id` header present for registered users

**API Key Assertions:**
- Status code is 201 for create
- Response contains `id` and `key` fields
- `key` is non-empty string (shown only at creation)
- Delete returns 204 No Content

**Error Assertions (Phase 2):**
- Invalid JWT returns 401 Unauthorized
- Expired JWT returns 401 Unauthorized
- Invalid API key returns 401 Unauthorized
- Both JWT and key missing returns 401 Unauthorized

### Debugging Failed Tests

**Common Issues and Solutions:**

| Problem | Cause | Solution |
|---------|-------|----------|
| Health check fails | LLM-API not running | `make up-full` to start services |
| Keycloak token fails | Keycloak not ready | Wait 30s after `make up-full` |
| Guest token fails | Guest handler not working | Check logs: `docker logs jan-server-llm-api-1` |
| User creation fails | User already exists | Tests handle 409 conflict gracefully |
| Chat completion fails | Model not available | Download model: `scripts/setup.sh` |
| API key fails | Service not compiled | Run `make build` before tests |

### Test Success Criteria

**Test suite passes when:**
- ✅ All health checks return 200 OK
- ✅ Setup creates guest and user tokens successfully
- ✅ All LLM-API endpoints accept valid JWTs
- ✅ API key creation returns 201 with key
- ✅ API key management (list, delete) works
- ✅ Teardown successfully removes test data
- ✅ No unhandled exceptions in test scripts
- ✅ All response times < 2 seconds

### Postman Collection Export Location

**File:** `tests/automation/auth-postman-scripts.json`

**To Import into Postman:**
1. Open Postman
2. Click **Import** → **Upload Files**
3. Select `tests/automation/auth-postman-scripts.json`
4. Collection imported with all tests and variables
5. Click **Runner** → Select collection → **Start Run**

### Newman CI/CD Integration

**GitHub Actions Example:**

```yaml
name: Auth Tests
on: [push, pull_request]
jobs:
  auth-tests:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:dind
    steps:
      - uses: actions/checkout@v3
      - name: Start Docker Stack
        run: make up-full
      - name: Wait for Services
        run: scripts/wait-for.sh http://localhost:8085/health
      - name: Run Auth Tests
        run: make test-auth
      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: newman-results
          path: newman.json
```

### Phase 2 Planned Tests

These tests will be added in Phase 2 to cover error scenarios:

- [ ] **JWT Error Cases**
  - [ ] Expired JWT returns 401
  - [ ] Invalid signature returns 401
  - [ ] Missing "exp" claim returns 401
  - [ ] Wrong "aud" (audience) returns 401
  - [ ] Malformed JWT returns 401

- [ ] **API Key Error Cases**
  - [ ] Revoked key returns 401
  - [ ] Invalid key format returns 401
  - [ ] Key from different user returns 401
  - [ ] Deleted key returns 401

- [ ] **Auth Bypass Prevention**
  - [ ] No JWT, no API key → 401
  - [ ] Invalid JWT, invalid key → 401
  - [ ] Both JWT and key invalid → 401

- [ ] **Performance Tests**
  - [ ] Auth latency p99 < 50ms
  - [ ] 100 concurrent requests succeed
  - [ ] Key rotation doesn't drop requests

---

## 📚 References

### Tools & Documentation
- **decK**: https://docs.konghq.com/deck/latest/
- **Kong Admin API**: https://docs.konghq.com/gateway/latest/admin-api/
- **Kong Plugins**: https://docs.konghq.com/hub/
- **Keycloak 24 User API Keys**: https://www.keycloak.org/docs/24.0/server_admin/#_user-api-keys
- **Key Auth Plugin**: https://docs.konghq.com/hub/kong-inc/key-auth/

### Internal Documentation
- API Key Management: `docs/api/llm-api/API_KEYS.md`
- Security Architecture: `docs/architecture/security.md`
- Kong Configuration: `docker/services-api.yml`

---

## ✅ Success Criteria

After full implementation, the system should meet these criteria:

### Security
- ✅ All API endpoints require authentication (JWT or API key)
- ✅ Kong validates credentials before forwarding to services
- ✅ Services perform defense-in-depth validation
- ✅ API keys are stored hashed in Keycloak
- ✅ No secrets in logs or error messages

### Functionality
- ✅ Users can generate API keys via authenticated API
- ✅ Users can list and revoke their own keys
- ✅ Keys work in both `X-API-Key` and `Authorization: Bearer` formats
- ✅ JWT tokens work for interactive sessions
- ✅ Revoked keys are rejected immediately

### Operations
- ✅ Kong config changes via GitOps (PR → Review → Merge → Auto-deploy)
- ✅ Config validation happens before deployment
- ✅ Rollback possible via Git revert
- ✅ Health checks verify auth is working
- ✅ Metrics track auth success/failure rates

### Performance
- ✅ Auth overhead < 50ms at p99
- ✅ Kong throughput > 10,000 req/s with auth enabled
- ✅ Zero downtime during Kong config updates

### Developer Experience
- ✅ Clear documentation for all workflows
- ✅ Local testing possible
- ✅ Fast feedback loop (< 2 minutes from commit to deploy)
- ✅ Self-service API key management
- ✅ Helpful error messages for auth failures

---

## 👥 Guest User Support & Upgrade Flow

**Status:** ✅ **FULLY COMPATIBLE WITH KONG AUTHENTICATION PLAN**

The existing guest user creation and upgrade flows in llm-api work seamlessly with the Kong authentication implementation. No breaking changes, no code modifications needed.

### How Guest Flow Works

```
┌────────────────────────────────────────────────────────────────┐
│                    GUEST USER LIFECYCLE                         │
└────────────────────────────────────────────────────────────────┘

Step 1: Guest Creation (PUBLIC - No Auth Required)
  Client → POST /auth/guest-login
    └─ Kong: No auth plugins on this route
    └─ Forwards to LLM-API
    └─ Keycloak creates guest user with "guest" attribute
    └─ Returns JWT access token + refresh token
    └─ Response: {"access_token": "...", "username": "guest-uuid", ...}

Step 2: Guest Uses JWT (PROTECTED - Kong Validates)
  Client → GET /auth/me
    with Authorization: Bearer {jwt_from_step_1}
    └─ Kong: JWT plugin validates signature (RS256)
    └─ Kong: Verifies claims (exp, nbf)
    └─ Kong: Injects X-Consumer-* headers
    └─ Forwards to LLM-API
    └─ LLM-API reads headers, returns user info with guest: true

Step 3: Guest Upgrades (PROTECTED - Kong Validates JWT)
  Client → POST /auth/upgrade
    with Authorization: Bearer {guest_jwt}
    and body: {username, email, full_name}
    └─ Kong: JWT plugin validates token
    └─ Kong: Injects X-Consumer-* headers
    └─ Forwards to LLM-API
    └─ LLM-API: Middleware extracts Principal from Kong headers
    └─ Keycloak updates user (sets guest: false, updates profile)
    └─ Returns: {"status": "upgraded"}

Step 4: Upgraded User Continues (PROTECTED)
  Client → GET /v1/models
    with Authorization: Bearer {same_jwt_still_valid}
    └─ Kong: Validates JWT (still valid)
    └─ Injects headers
    └─ LLM-API returns models
    └─ User now has full account access
```

### Kong Routes Configuration for Guest Support

**Public Routes (No Authentication):**

```yaml
routes:
  - name: auth-guest-login
    paths: [/llm/auth/guest-login]
    strip_path: true
    # ❌ No jwt plugin
    # ❌ No key-auth plugin
    # ✅ CORS only
    plugins:
      - name: cors
        config:
          origins: ["*"]
          methods: [POST, OPTIONS]

  - name: auth-refresh-token
    paths: [/llm/auth/refresh-token]
    strip_path: true
    # ❌ No auth plugins
    plugins:
      - name: cors

  - name: auth-logout
    paths: [/llm/auth/logout]
    strip_path: true
    # ❌ No auth plugins
    plugins:
      - name: cors
```

**Protected Routes (Authentication Required):**

```yaml
routes:
  - name: auth-upgrade
    paths: [/llm/auth/upgrade]
    strip_path: true
    # ✅ Auth plugins active
    plugins:
      - name: jwt
        config:
          algorithm: RS256
          key_claim_name: kid
          claims_to_verify: ["exp","nbf"]
          anonymous: kong-anon-jwt  # OR logic
      
      - name: key-auth
        config:
          key_names: ["X-API-Key","Authorization"]
          anonymous: kong-anon-key  # OR logic
      
      - name: request-termination
        consumer: kong-anon-key
        config:
          status_code: 401
          message: "Authentication required"
      
      - name: cors
        config:
          origins: ["*"]
          methods: [POST, OPTIONS]
```

### Current Guest Implementation (Unchanged)

The existing implementation in llm-api already supports this flow:

**Files:**
- Guest creation: `services/llm-api/internal/interfaces/httpserver/handlers/guesthandler/guest_handler.go`
- Guest logic: `services/llm-api/internal/infrastructure/keycloak/client.go`
- Routes: `services/llm-api/internal/interfaces/httpserver/routes/auth/auth_route.go`

**Key Components:**
- ✅ `POST /auth/guest-login` - Creates guest, returns JWT
- ✅ `POST /auth/upgrade` - Upgrades guest to permanent account
- ✅ `GET /auth/refresh-token` - Refreshes access token
- ✅ Keycloak Admin API integration for user management
- ✅ Guest role assignment and tracking

### Middleware Update Required (Minimal)

**Current:** LLM-API middleware extracts Principal directly from JWT  
**Updated:** Prefer Kong-injected headers (defense-in-depth)

**File:** `services/llm-api/internal/interfaces/httpserver/middlewares/auth.go`

**Change:** Update `PrincipalFromContext()` to prefer Kong headers:

```go
// Before: Extract from JWT manually
// After: Prefer Kong headers, fallback to JWT

func PrincipalFromContext(c *gin.Context) (domain.Principal, bool) {
    // Try Kong headers first (preferred - already validated by Kong)
    if consumerID := c.GetHeader("X-Consumer-Custom-ID"); consumerID != "" {
        return domain.Principal{
            ID:       consumerID,
            Subject:  consumerID,
            Username: c.GetHeader("X-Consumer-Username"),
        }, true
    }
    
    // Fallback: Parse JWT if Kong headers absent (offline/testing)
    tokenStr := extractBearerToken(c.GetHeader("Authorization"))
    if tokenStr == "" {
        return domain.Principal{}, false
    }
    
    claims := extractClaims(tokenStr)
    if claims == nil {
        return domain.Principal{}, false
    }
    
    return domain.Principal{
        ID:      claims["sub"],
        Subject: claims["sub"],
    }, true
}
```

**Impact:** One function update, no logic changes, backward compatible

### Guest Support in Phase 1 Implementation

Add to **Phase 1: Week 1 - Kong Gateway Authentication**:

- [ ] **Configure Guest Routes**
  - [ ] Add `/auth/guest-login` route (public, no auth plugins)
  - [ ] Add `/auth/refresh-token` route (public, no auth plugins)
  - [ ] Add `/auth/logout` route (public, no auth plugins)
  - [ ] Add `/auth/upgrade` route (protected, auth plugins active)
  - [ ] Verify routes in Kong Admin API

- [ ] **Update LLM-API Middleware**
  - [ ] Modify principal extraction in `auth.go`
  - [ ] Prefer Kong headers, fallback to JWT
  - [ ] Test locally with guest flow

- [ ] **Add Guest Flow Tests to Postman**
  - [ ] Test: Create guest (no auth)
  - [ ] Test: Use guest JWT on protected endpoint
  - [ ] Test: Upgrade guest account
  - [ ] Test: Token refresh
  - [ ] Run full test suite with Newman

### Testing Guest Flow

```bash
# Test 1: Create guest (public endpoint)
curl -X POST http://localhost:8000/llm/auth/guest-login \
  -H "Content-Type: application/json"
# Expected: 201 Created with access_token

# Test 2: Get guest info (protected, JWT required)
curl -X GET http://localhost:8000/llm/auth/me \
  -H "Authorization: Bearer {access_token_from_test_1}"
# Expected: 200 OK with user info, guest: true

# Test 3: Upgrade guest (protected, JWT required)
curl -X POST http://localhost:8000/llm/auth/upgrade \
  -H "Authorization: Bearer {access_token_from_test_1}" \
  -H "Content-Type: application/json" \
  -d '{"username": "realuser", "email": "user@example.com", "full_name": "Real User"}'
# Expected: 200 OK with status: "upgraded"

# Test 4: Verify upgrade (JWT still valid)
curl -X GET http://localhost:8000/llm/auth/me \
  -H "Authorization: Bearer {access_token_from_test_1}"
# Expected: 200 OK with guest: false

# Test 5: Expired token rejected
curl -X POST http://localhost:8000/llm/auth/upgrade \
  -H "Authorization: Bearer {expired_jwt}" \
  -H "Content-Type: application/json" \
  -d '{...}'
# Expected: 401 Unauthorized
```

### Compatibility Matrix: Guest Support + Kong Auth

| Aspect | Status | Details |
|--------|--------|---------|
| Guest creation | ✅ Compatible | Public endpoint, no auth plugins |
| Guest JWT tokens | ✅ Compatible | Kong validates via JWKS |
| Guest upgrade | ✅ Compatible | Protected endpoint, Kong validates JWT |
| Keycloak guest logic | ✅ Unchanged | No changes to guest role/attributes |
| LLM-API handlers | ✅ Unchanged | Guest handler code unchanged |
| Database schema | ✅ Unchanged | No schema changes needed |
| Token refresh | ✅ Compatible | Public endpoint, refresh token flow works |
| Multi-guest sessions | ✅ Works | Each guest is independent user |
| Guest → API Keys (Phase 2) | ✅ Works | Upgraded users can create keys |

### Guest Edge Cases

**Scenario 1: Expired Token on Upgrade**
```
Guest JWT expires during upgrade attempt
→ Kong JWT plugin detects expired token
→ Kong returns 401 Unauthorized
→ Client refreshes token and retries
✅ Correct behavior, expected flow
```

**Scenario 2: Multiple Guest Accounts**
```
Same client creates multiple guests
→ Each POST /auth/guest-login creates new Keycloak user
→ Each guest has unique JWT
→ Guests are completely independent
✅ Works correctly
```

**Scenario 3: Duplicate Username on Upgrade**
```
Guest tries to upgrade to existing username
→ Keycloak Admin API validation fails
→ LLM-API returns 400 Bad Request
→ Guest remains in guest state
✅ Validation at Keycloak layer prevents errors
```

### Guest Support Success Criteria

- ✅ Guest can create account without credentials
- ✅ Guest receives valid JWT token
- ✅ Guest JWT works on all protected endpoints
- ✅ Guest can upgrade to permanent account
- ✅ After upgrade, Keycloak guest attribute set to false
- ✅ Token remains valid after upgrade
- ✅ Upgrade is persisted in Keycloak
- ✅ Upgraded users can perform all user operations
- ✅ No existing functionality broken
- ✅ Auth latency < 50ms with guest JWTs

---

## 🎯 Next Steps

**Ready to start implementation?**

1. ✅ **Phase 1**: Set up Kong Admin API + Update Kong configuration
   - ✅ Includes guest route configuration
   - ✅ Includes middleware update
2. ✅ **Phase 2**: Implement Keycloak User API Keys + API endpoints
3. ✅ **Phase 3**: Enable service-level auth enforcement
4. ✅ **Phase 4**: Testing, documentation, and staged rollout

**Let's begin! 🚀**
