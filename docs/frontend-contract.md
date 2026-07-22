# Frontend Contract

## Purpose

This document defines how the React frontend should interact with the backend.

## API Base

```text
/api/v1
```

## Response Shape

Success:

```ts
type ApiSuccess<T, M = unknown> = {
  data: T
  meta?: M
}
```

Error:

```ts
type ApiError = {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}
```

## Auth Contract

Frontend should:

- Call `POST /api/v1/auth/login`.
- Store access token only short-term.
- Rely on secure refresh cookie if implemented.
- Call `POST /api/v1/auth/refresh` on access-token expiry.
- Clear local auth state on logout.

## Current User

```text
GET /api/v1/me
```

Expected data:

```ts
type CurrentUser = {
  id: string
  email?: string
  status: string
  roles: string[]
  profile: Profile
  student_identity?: StudentIdentity
}
```

## Roles

```ts
type Role =
  | 'student'
  | 'student_coordinator'
  | 'faculty'
  | 'hod'
  | 'placement_officer'
  | 'principal'
  | 'alumni'
  | 'club_organizer'
  | 'admin'
```

## Dashboards

Frontend should render dashboards based on roles:

- Student dashboard
- Coordinator dashboard
- Faculty dashboard
- HOD dashboard
- Placement officer dashboard
- Principal dashboard
- Admin dashboard

If a user has multiple roles, show a role switcher or grouped sections.

## Visibility Rules

Frontend can hide buttons for UX, but backend is the source of truth.

Examples:

- Hide event approval button for non-HOD users.
- Hide shortlisting tools for non-placement users.
- Hide admin user-management screens for non-admin users.

Backend must still enforce every permission.

## Pagination Contract

```ts
type PageMeta = {
  next_cursor?: string
}
```

## Application Statuses

```ts
type OpportunityApplicationStatus =
  | 'saved'
  | 'applied'
  | 'shortlisted'
  | 'interview'
  | 'selected'
  | 'rejected'
  | 'withdrawn'
  | 'closed'
```

## Event Statuses

```ts
type EventApprovalStatus =
  | 'draft'
  | 'submitted'
  | 'hod_change_requested'
  | 'hod_rejected'
  | 'hod_approved'
  | 'final_change_requested'
  | 'final_rejected'
  | 'published'
  | 'cancelled'
```

## Forms

Forms should match backend DTOs and show backend validation errors inline.

Important forms:

- Access request
- Profile edit
- Announcement create/edit
- Event draft
- HOD review
- Final event approval
- Opportunity create/edit
- Application status update
- CSV import

## File Uploads

Frontend should validate before upload:

- MIME type
- File size
- Image dimensions if required later

Backend remains the final authority.

## Notifications

Frontend should support:

- In-app notification list
- Unread count
- Mark notification as read
- Mark all notifications as read
- Push subscription registration after user permission
- Notification preferences

Important APIs:

```text
GET    /api/v1/notifications
PATCH  /api/v1/notifications/:id/read
PATCH  /api/v1/notifications/read-all
GET    /api/v1/notification-preferences
PATCH  /api/v1/notification-preferences
POST   /api/v1/push-subscriptions
DELETE /api/v1/push-subscriptions/:id
```
