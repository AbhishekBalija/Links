# Postman Test Guide — Auth Flow

Base URL: `http://localhost:8080`

---

## 1. Request Access

Creates a new user with `pending` status.

**Request**

```
POST /api/v1/auth/request-access
Content-Type: application/json

{
  "email": "test@mitt.edu.in",
  "password": "TestPass123",
  "full_name": "Test User",
  "usn": "4XX21XX001",
  "department_code": "CS"
}
```

**Success Response — `201 Created`**

```json
{
  "data": {
    "id": "473b87ac-cc4a-4366-85bd-709d99830407",
    "status": "pending"
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| Missing required field | 400 | `VALIDATION_ERROR` |
| Email already registered | 409 | `CONFLICT` |
| Invalid department code | 400 | `VALIDATION_ERROR` |

---

## 2. Activate User

The activation email is sent via Resend on sign-up. The `/activate` endpoint sets the user's password
and flips status to `active`.

**Request:** `POST /api/v1/auth/activate`

```json
{
  "token": "<raw_activation_token_from_email>",
  "password": "MySecurePass123"
}
```

**Response:** `200`

```json
{
  "data": {
    "message": "account activated"
  }
}
```

**Note:** In local dev without a `RESEND_API_KEY`, no email is sent. To activate manually:

```sql
UPDATE users SET status = 'active', is_verified = true WHERE email = 'test@mitt.edu.in';
```

After activation, assign a role:

```sql
INSERT INTO role_assignments (id, user_id, role, scope_type, scope_id, assigned_by, starts_at, created_at)
VALUES (gen_random_uuid(), '<user_id>', 'student', 'global', NULL, NULL, now(), now());
```

Available roles: `student`, `student_coordinator`, `faculty`, `hod`, `placement_officer`, `principal`, `alumni`, `club_organizer`, `admin`

---

## 3. Login

Authenticates and returns a JWT access token + HTTP-only refresh cookie.

**Request**

```
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "test@mitt.edu.in",
  "password": "TestPass123"
}
```

**Success Response — `200 OK`**

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

The `refresh_token` is set as an `HttpOnly` cookie on the response.

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| Wrong email/password | 401 | `UNAUTHENTICATED` |
| Account not active (pending/suspended/rejected) | 401 | `UNAUTHENTICATED` |

---

## 4. Get Current User

Returns the authenticated user's info. Requires the JWT from login.

**Request**

```
GET /api/v1/me
Authorization: Bearer <access_token>
```

**Success Response — `200 OK`**

```json
{
  "data": {
    "id": "473b87ac-cc4a-4366-85bd-709d99830407",
    "email": "test@mitt.edu.in",
    "roles": ["student"]
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| No token provided | 401 | `UNAUTHENTICATED` |
| Invalid/expired token | 401 | `UNAUTHENTICATED` |

---

## 5. Refresh Tokens

Rotates the refresh token. Requires the `refresh_token` cookie from login.

**Request**

```
POST /api/v1/auth/refresh
```

No request body — reads the `refresh_token` cookie automatically.

**Success Response — `200 OK`**

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

A new `refresh_token` cookie is set (rotation).

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| No refresh cookie | 401 | `UNAUTHENTICATED` |
| Expired refresh token | 401 | `UNAUTHENTICATED` |
| Revoked refresh token | 401 | `UNAUTHENTICATED` |

---

## 6. Logout

Revokes the refresh token and clears the cookie.

**Request**

```
POST /api/v1/auth/logout
```

No request body — reads the `refresh_token` cookie automatically.

**Success Response — `200 OK`**

```json
{
  "data": {
    "message": "logged out successfully"
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| No refresh cookie | 401 | `UNAUTHENTICATED` |
| Refresh token already revoked | 401 | `UNAUTHENTICATED` |

---

## 7. Admin Auth Setup

Before testing admin endpoints, bootstrap the first admin user via SQL:

```sql
INSERT INTO users (id, email, password_hash, status, is_verified, created_at, updated_at)
VALUES (gen_random_uuid(), 'admin@mitt.edu.in',
  '$argon2id$v=19$m=32768,t=1,p=2$7LgbXRLytdp7E7dNkqyexw$ekGNwq+3hZxmyAm/kCgLFzKoiCOtL/Fia3XE9+GXIe8',
  'active', true, now(), now());

-- Get the user_id from the insert above, then:
INSERT INTO role_assignments (id, user_id, role, scope_type, scope_id, assigned_by, starts_at, created_at)
VALUES (gen_random_uuid(), '<user_id>', 'admin', 'global', NULL, NULL, now(), now());

INSERT INTO profiles (user_id, username, full_name, created_at, updated_at)
VALUES ('<user_id>', 'admin', 'Admin', now(), now());
```

Then login as the local admin using the email and password you chose for this
bootstrap. Do not use shared or production credentials in this guide:

```
POST /api/v1/auth/login
Content-Type: application/json

{"email": "<LOCAL_ADMIN_EMAIL>", "password": "<LOCAL_ADMIN_PASSWORD>"}
```

Copy the `access_token` from the response for all admin endpoint calls.

---

## 8. Admin: Review Queue

Lists all pending users. Requires `admin` or `principal` role.

**Request**

```
GET /api/v1/admin/users/review-queue
Authorization: Bearer <admin_access_token>
```

**Success Response — `200 OK`**

```json
{
  "data": {
    "users": [
      {
        "id": "473b87ac-cc4a-4366-85bd-709d99830407",
        "email": "test@mitt.edu.in",
        "profile": {
          "full_name": "Test User",
          "username": "test.user1234"
        },
        "student_identity": {
          "usn": "4XX21XX001",
          "batch_year": 2024
        },
        "created_at": "2026-07-21T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| Insufficient role | 403 | `FORBIDDEN` |
| Not authenticated | 401 | `UNAUTHENTICATED` |

---

## 9. Admin: Verify User

Approves a pending user: assigns `student` role, flips status to `active`, creates audit log.

**Request**

```
PATCH /api/v1/admin/users/:id/verify
Authorization: Bearer <admin_access_token>
Content-Type: application/json

{
  "note": "Verified via manual approval"
}
```

Optional fields: `scope_type` (defaults to `global`), `scope_id`.

**Success Response — `200 OK`**

```json
{
  "data": {
    "message": "user verified successfully"
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| User not found | 404 | `NOT_FOUND` |
| User not in pending status | 409 | `CONFLICT` |
| Insufficient role | 403 | `FORBIDDEN` |

---

## 10. Admin: Update User Status

Rejects, suspends, or restores a user. Cannot activate a previously rejected user.

**Request**

```
PATCH /api/v1/admin/users/:id/status
Authorization: Bearer <admin_access_token>
Content-Type: application/json

{
  "status": "rejected",
  "note": "Invalid USN"
}
```

Valid status values: `active`, `suspended`, `rejected`.

**Success Response — `200 OK`**

```json
{
  "data": {
    "message": "user status updated successfully"
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| Invalid status value | 400 | `VALIDATION_ERROR` |
| User not found | 404 | `NOT_FOUND` |
| Same status (no-op) | 409 | `CONFLICT` |
| Activate rejected user | 400 | `VALIDATION_ERROR` |
| Insufficient role | 403 | `FORBIDDEN` |

---

---

## 11. Resend Activation

Requests a new activation email. Silently returns success for non-existent or already-active emails (enumeration protection).

**Request**

```
POST /api/v1/auth/resend-activation
Content-Type: application/json

{
  "email": "pending@mitt.edu.in"
}
```

**Success Response — `200 OK`** (identical for registered-pending, registered-active, and non-existent emails)

```json
{
  "data": {
    "message": "activation email sent"
  }
}
```

**Error Responses**

| Scenario | Status | Code |
|---|---|---|
| Resend within 5 minutes of last send | 429 | `RATE_LIMITED` |

---

## 12. Activation Error Responses

All activation failure modes return exactly **401 Unauthorized** with a generic message (no enumeration):

| Scenario | Status | Response |
|---|---|---|
| Invalid token (no match) | 401 | `{"error":{"code":"UNAUTHENTICATED","message":"invalid or expired activation token"}}` |
| Expired token | 401 | Same as above |
| Already-used token | 401 | Same as above |
| Malformed/short token | 401 | Same as above |
| SQL injection / XSS in token | 401 | Same as above |

---

## 13. Live E2E Verification (2026-07-22)

Full pipeline tested against Neon DB + Resend production API.

| Step | Endpoint | Result |
|---|---|---|
| 1 | `POST /request-access` | `201` — user created, status `pending` |
| 2 | DB: `account_activation_tokens` | Token inserted, `used_at` null, 7-day expiry |
| 3 | Resend API | Email accepted and delivered to inbox |
| 4 | Extract raw token from email link | Token extracted from URL query param |
| 5 | `POST /activate` with raw token | `200` — `"account activated"` |
| 6 | DB: `users` | `status=active`, `is_verified=true` |
| 7 | `POST /login` with email + password | `200` — access token + refresh cookie |
| 8 | `POST /resend-activation` (active user) | `200` — silently returns success |
| 9 | `POST /resend-activation` (pending, cooldown expired) | Old token revoked, new token created, email sent |
| 10 | `POST /resend-activation` (immediate second attempt) | `429 RATE_LIMITED` — 5-min cooldown enforced |

### Resend Sandbox Note

Resend's `onboarding@resend.dev` sandbox sender only delivers to the Resend account owner's email.  
Production with a custom domain (`noreply@<domain>`) has no such restriction.  
Gmail `+` aliases (e.g. `user+tag@gmail.com`) are rejected by the sandbox API — emails must use the exact owner address.

### Activation Token Format

| Property | Value |
|---|---|
| Source | 32 bytes from `crypto/rand` |
| Encoding | `base64.RawURLEncoding` (no padding) |
| Storage | SHA-256 hash in `account_activation_tokens.token_hash` |
| Link | `FRONTEND_URL/activate?token=<raw>` |
| Expiry | 7 days from creation |

---

## Full E2E Test Flow (Postman Collection Order)

1. **Request Access** → copy `user_id` from response
2. **Activate via SQL** → run SQL queries above
3. **Login** → copy `access_token`
4. **Get Me with token** → verify roles match
5. **Refresh** → new token issued
6. **Get Me with new token** → still works
7. **Logout** → token revoked
8. **Get Me without token** → `401`
9. **Admin: Review Queue** → verify pending user appears
10. **Admin: Verify User** → approve the pending user
11. **Admin: Update User Status** → test reject/suspend/restore
