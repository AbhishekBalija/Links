# Postman Test Guide: Auth Flow

Base URL: `http://localhost:8080`

---

## 1. Ask for a sign-in code

There are no passwords (ADR 0026). Members sign in with Google or a one-time
email code; Google needs a browser, so use the email code here.

```http
POST /api/v1/auth/code
Content-Type: application/json

{"email": "<A_MEMBER_EMAIL>"}
```

Always `200` with a `challenge_id`, whether or not the email is known. The
code arrives by email (Resend). Without a Resend key, run the server with
`ENABLE_TEST_SIGN_IN=true` and `APP_ENV=local` and read it back:

```http
GET /api/v1/test/sign-in-code?email=<A_MEMBER_EMAIL>
```

## 2. Sign in with the code

```http
POST /api/v1/auth/code/verify
Content-Type: application/json

{"challenge_id": "<from step 1>", "email": "<A_MEMBER_EMAIL>", "code": "123456"}
```

`200` with `access_token` and the `refresh_token` cookie. An email on no list
gets `403 NOT_ON_LIST` with a `request_token` for `POST
/api/v1/auth/access-request`; see `docs/api-spec.md`, Auth.

## 3. Sign in without a code (local only)

With `ENABLE_TEST_SIGN_IN=true` and `APP_ENV=local`, an active member can get
a session by email alone:

```http
POST /api/v1/test/sign-in
Content-Type: application/json

{"email": "<A_MEMBER_EMAIL>"}
```

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

Before testing admin endpoints, add the first admin from the command line,
with the server's environment loaded (ADR 0026):

```bash
cd server
go run ./cmd/add-admin -email <YOUR_GOOGLE_EMAIL> -name "Local Admin"
```

It refuses if the database already has an admin. The admin has no password:
sign in once at the web app (`http://localhost:5173`) with **Continue with
Google** using that email, which makes the account active. After that, with
`ENABLE_TEST_SIGN_IN=true` and `APP_ENV=local`, get a token for Postman
without Google:

```http
POST /api/v1/test/sign-in
Content-Type: application/json

{"email": "<YOUR_GOOGLE_EMAIL>"}
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

## Resend Sandbox Note

Resend's `onboarding@resend.dev` sandbox sender only delivers to the Resend account owner's email.  
Production with a custom domain (`noreply@<domain>`) has no such restriction.  
Gmail `+` aliases (e.g. `user+tag@gmail.com`) are rejected by the sandbox API — emails must use the exact owner address.

---

## Full E2E Test Flow (Postman Collection Order)

1. **Ask for a code** → copy `challenge_id`
2. **Read the code** (`/api/v1/test/sign-in-code`, local) or from the email
3. **Sign in with the code** → copy `access_token`
4. **Get Me with token** → verify roles match
5. **Refresh** → new token issued
6. **Get Me with new token** → still works
7. **Logout** → token revoked
8. **Get Me without token** → `401`
9. **Admin: Review Queue** → verify pending user appears
10. **Admin: Verify User** → approve the pending user
11. **Admin: Update User Status** → test reject/suspend/restore
