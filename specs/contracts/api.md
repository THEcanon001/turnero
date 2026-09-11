# API Contracts: Turnero App

---

## Conventions

| Convention | Value |
|------------|-------|
| Base URL | `https://api.tuapp.com/v1` |
| Auth | Bearer token (JWT) in `Authorization` header |
| Content-Type | `application/json` |
| Pagination | `?page=1&per_page=20`, response header `X-Total-Count` |
| Date format | `YYYY-MM-DD` (ISO 8601) |
| Time format | `HH:MM` (24h) |
| Timezone | All times in provider's configured timezone. Stored as UTC in DB. |
| IDs | UUID v4 |

### Error Response Format

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {}  // Optional: field-level validation errors
}
```

### Common Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `VALIDATION_ERROR` | Request body fails validation |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT |
| 401 | `TOKEN_EXPIRED` | JWT has expired |
| 403 | `FORBIDDEN` | Authenticated but lacks permission |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Resource conflict (e.g., slug taken, slot taken) |
| 409 | `SLOT_TAKEN` | Appointment slot already booked |
| 422 | `BUSINESS_RULE_VIOLATION` | Business rule prevents action (e.g., cancel too late) |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Server error |

---

## Rate Limits

| Endpoint Group | Limit | Scope |
|----------------|-------|-------|
| `POST /auth/login` | 5 req/min | Per IP |
| `POST /auth/register` | 3 req/min | Per IP |
| `POST /appointments` | 10 req/min | Per IP |
| `GET /search` | 30 req/min | Per IP |
| Authenticated endpoints (general) | 60 req/min | Per user |
| `GET /appointments` | 120 req/min | Per user |

---

## Endpoints

### 1. Auth (Public)

#### POST /v1/auth/register

Register a new provider (PF or PJ).

**Auth:** None

**Request:**
```json
{
  "name": "Juan Pérez",
  "email": "juan@example.com",
  "password": "securepassword123",
  "phone": "+521234567890",
  "type": "individual",
  "slug": "barberia-juan",
  "business_name": null,
  "timezone": "America/Mexico_City"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| name | string | yes | min 2, max 100 |
| email | string | yes | valid email, unique |
| password | string | yes | min 8 |
| phone | string | yes | valid phone format |
| type | enum | yes | `"individual"` or `"business"` |
| slug | string | yes | unique, lowercase, only `[a-z0-9-]`, min 3, max 50 |
| business_name | string | no | required if type = `"business"` |
| timezone | string | no | valid IANA timezone, default `"America/Mexico_City"` |

**Response (201 Created):**
```json
{
  "id": "prov-uuid-123",
  "name": "Juan Pérez",
  "email": "juan@example.com",
  "type": "individual",
  "slug": "barberia-juan",
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh..."
}
```

**Errors:**
- `409 CONFLICT` — email or slug already taken

---

#### POST /v1/auth/login

**Auth:** None

**Request:**
```json
{
  "email": "juan@example.com",
  "password": "securepassword123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "prov-uuid-123",
    "name": "Juan Pérez",
    "role": "admin",
    "type": "individual",
    "provider_id": "prov-uuid-123"
  }
}
```

**Errors:**
- `401 UNAUTHORIZED` — invalid email or password

---

#### POST /v1/auth/refresh

**Auth:** None

**Request:**
```json
{
  "refresh_token": "dGhpcyBpcyBh..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "bmV3IHJlZnJlc2g...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

Token rotation: the old refresh token is revoked, a new pair is issued.

**Errors:**
- `401 UNAUTHORIZED` — invalid or revoked refresh token

---

#### POST /v1/auth/logout

**Auth:** None (refresh token in body)

**Request:**
```json
{
  "refresh_token": "dGhpcyBpcyBh..."
}
```

**Response (204 No Content)**

---

### 2. Provider (Authenticated)

#### GET /v1/provider/me

**Auth:** Bearer (provider or employee)

**Response (200 OK):**
```json
{
  "id": "prov-uuid-123",
  "type": "individual",
  "name": "Juan Pérez",
  "slug": "barberia-juan",
  "phone": "+521234567890",
  "address": "Calle Falsa 123, CDMX",
  "timezone": "America/Mexico_City",
  "email": "juan@example.com",
  "plan": "free",
  "is_active": true,
  "created_at": "2026-09-01T10:00:00Z"
}
```

---

#### PUT /v1/provider/me

**Auth:** Bearer (provider admin)

**Request:**
```json
{
  "name": "Juan Pérez - Barbería",
  "phone": "+521234567890",
  "address": "Calle Falsa 123, CDMX",
  "timezone": "America/Mexico_City"
}
```

**Response (200 OK):** Updated provider object.

---

#### GET /v1/provider/me/qr

**Auth:** Bearer (provider admin)

**Response (200 OK):** PNG image (`Content-Type: image/png`)

QR encodes: `https://tuapp.com/p/{slug}`

---

#### GET /v1/provider/me/billing

**Auth:** Bearer (provider admin)

**Response (200 OK):**
```json
{
  "month": "2026-09-01",
  "completed_appointments": 23,
  "free_tier_limit": 20,
  "billable_appointments": 3,
  "amount_due": 1.50,
  "cap_amount": 10.00,
  "currency": "USD",
  "is_paid": false
}
```

---

#### GET /v1/provider/me/billing/history

**Auth:** Bearer (provider admin)

**Query params:** `?page=1&per_page=12`

**Response (200 OK):**
```json
{
  "items": [
    {
      "month": "2026-08-01",
      "completed_appointments": 45,
      "billable_appointments": 25,
      "amount_due": 10.00,
      "is_paid": true
    }
  ],
  "total": 3,
  "page": 1,
  "per_page": 12
}
```

---

### 3. Services (Authenticated — Provider)

#### GET /v1/services

**Auth:** Bearer (provider)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "svc-uuid-1",
      "name": "Corte de pelo",
      "description": "Corte clásico",
      "duration_minutes": 30,
      "is_active": true
    }
  ]
}
```

---

#### POST /v1/services

**Auth:** Bearer (provider admin)

**Request:**
```json
{
  "name": "Corte de pelo",
  "description": "Corte clásico",
  "duration_minutes": 30
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| name | string | yes | min 2, max 100 |
| description | string | no | max 500 |
| duration_minutes | int | yes | 5-480 |

**Response (201 Created):** Service object.

---

#### PUT /v1/services/:id

**Auth:** Bearer (provider admin)

**Request:** Same as POST (partial update).

**Response (200 OK):** Updated service object.

---

#### DELETE /v1/services/:id

**Auth:** Bearer (provider admin)

Soft delete (sets `is_active = false`).

**Errors:**
- `422 BUSINESS_RULE_VIOLATION` — service has future appointments

**Response (204 No Content)**

---

### 4. Employees (Authenticated — Admin PJ)

#### GET /v1/employees

**Auth:** Bearer (provider admin)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "emp-uuid-1",
      "name": "María López",
      "phone": "+521234567891",
      "role": "employee",
      "is_active": true,
      "services": [
        {"id": "svc-uuid-1", "name": "Corte de pelo"}
      ]
    }
  ]
}
```

---

#### POST /v1/employees

**Auth:** Bearer (provider admin)

Creates invitation code for a new employee.

**Request:**
```json
{
  "name": "María López"
}
```

**Response (201 Created):**
```json
{
  "id": "emp-uuid-1",
  "name": "María López",
  "invitation_code": "ABC123XY",
  "invitation_expires_at": "2026-09-12T10:00:00Z",
  "invitation_link": "https://tuapp.com/join/ABC123XY"
}
```

---

#### GET /v1/employees/:id

**Auth:** Bearer (provider admin or the employee themselves)

**Response (200 OK):** Employee object with assigned services.

---

#### PUT /v1/employees/:id

**Auth:** Bearer (provider admin)

**Request:**
```json
{
  "name": "María López García",
  "service_ids": ["svc-uuid-1", "svc-uuid-2"]
}
```

**Response (200 OK):** Updated employee object.

---

#### DELETE /v1/employees/:id

**Auth:** Bearer (provider admin)

Soft delete (sets `is_active = false`).

**Response (200 OK):**
```json
{
  "deactivated": true,
  "pending_appointments": 5,
  "message": "Employee deactivated. 5 future appointments need to be reassigned or cancelled."
}
```

---

### 5. Schedules (Authenticated)

#### GET /v1/employees/:id/schedules

**Auth:** Bearer (provider admin or the employee)

**Response (200 OK):**
```json
{
  "employee_id": "emp-uuid-1",
  "schedules": [
    {
      "id": "sch-uuid-1",
      "day_of_week": 1,
      "day_name": "Monday",
      "start_time": "09:00",
      "end_time": "18:00",
      "slot_duration_minutes": 30,
      "break_after_slot_minutes": 0,
      "is_active": true
    }
  ]
}
```

---

#### PUT /v1/employees/:id/schedules

Batch upsert — replaces all schedules for the employee.

**Auth:** Bearer (provider admin or the employee)

**Request:**
```json
{
  "schedules": [
    {
      "day_of_week": 1,
      "start_time": "09:00",
      "end_time": "18:00",
      "slot_duration_minutes": 30,
      "break_after_slot_minutes": 0
    },
    {
      "day_of_week": 2,
      "start_time": "09:00",
      "end_time": "14:00",
      "slot_duration_minutes": 30
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "schedules": [...],
  "conflicts": [
    {
      "day_of_week": 1,
      "affected_appointments": 2,
      "message": "2 appointments fall outside the new schedule on Monday"
    }
  ]
}
```

If conflicts exist, the update still applies. Client should prompt user to handle affected appointments.

---

#### GET /v1/employees/:id/schedule-exceptions

**Auth:** Bearer (provider admin or the employee)

**Query params:** `?from=2026-09-01&to=2026-09-30`

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "exc-uuid-1",
      "date": "2026-09-16",
      "is_available": false,
      "reason": "Feriado nacional"
    },
    {
      "id": "exc-uuid-2",
      "date": "2026-09-20",
      "is_available": true,
      "start_time": "10:00",
      "end_time": "14:00",
      "reason": "Horario especial sábado"
    }
  ]
}
```

---

#### POST /v1/employees/:id/schedule-exceptions

**Auth:** Bearer (provider admin or the employee)

**Request:**
```json
{
  "date": "2026-09-16",
  "is_available": false,
  "reason": "Feriado nacional"
}
```

**Response (201 Created):**
```json
{
  "id": "exc-uuid-1",
  "date": "2026-09-16",
  "is_available": false,
  "reason": "Feriado nacional",
  "affected_appointments": [
    {
      "id": "apt-uuid-1",
      "client_name": "María García",
      "start_time": "10:00"
    }
  ]
}
```

---

#### DELETE /v1/employees/:id/schedule-exceptions/:eid

**Auth:** Bearer (provider admin or the employee)

**Response (204 No Content)**

---

### 6. Availability (Public — for Clients)

#### GET /v1/providers/:slug

Public profile of a provider/business.

**Auth:** None

**Response (200 OK):**
```json
{
  "slug": "barberia-juan",
  "name": "Barbería Juan",
  "type": "individual",
  "phone": "+521234567890",
  "address": "Calle Falsa 123, CDMX",
  "services": [
    {
      "id": "svc-uuid-1",
      "name": "Corte de pelo",
      "description": "Corte clásico",
      "duration_minutes": 30
    },
    {
      "id": "svc-uuid-2",
      "name": "Barba",
      "duration_minutes": 20
    }
  ],
  "employees": [
    {
      "id": "emp-uuid-1",
      "name": "Juan Pérez",
      "services": ["svc-uuid-1", "svc-uuid-2"]
    }
  ],
  "whatsapp_link": "https://wa.me/521234567890"
}
```

---

#### GET /v1/providers/:slug/employees

List available employees for a provider.

**Auth:** None

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "emp-uuid-1",
      "name": "Juan Pérez",
      "services": [
        {"id": "svc-uuid-1", "name": "Corte de pelo"}
      ]
    }
  ]
}
```

---

#### GET /v1/providers/:slug/employees/:eid/slots

Get available time slots for a specific employee on a given date or week.

**Auth:** None

**Query params:**
- `date=2026-09-15` — slots for a single day
- `week=2026-W38` — slots for a full week
- `service_id=svc-uuid-1` — filter by service duration (optional)

**Response (200 OK) — single day:**
```json
{
  "date": "2026-09-15",
  "employee": {
    "id": "emp-uuid-1",
    "name": "Juan Pérez"
  },
  "slots": [
    {"start": "09:00", "end": "09:30", "available": true},
    {"start": "09:30", "end": "10:00", "available": true},
    {"start": "10:00", "end": "10:30", "available": false},
    {"start": "10:30", "end": "11:00", "available": true},
    {"start": "11:00", "end": "11:30", "available": true},
    {"start": "11:30", "end": "12:00", "available": true}
  ]
}
```

**Response (200 OK) — week:**
```json
{
  "week": "2026-W38",
  "employee": {
    "id": "emp-uuid-1",
    "name": "Juan Pérez"
  },
  "days": [
    {
      "date": "2026-09-14",
      "day_name": "Monday",
      "slots": [...]
    },
    {
      "date": "2026-09-15",
      "day_name": "Tuesday",
      "slots": [...]
    }
  ]
}
```

Slot availability is computed from:
1. Employee's weekly schedule for that day_of_week
2. Minus schedule_exceptions (day off or special hours)
3. Minus existing confirmed appointments
4. Respecting service duration (multi-slot if service > slot_duration)

---

### 7. Appointments — Client (Public)

#### POST /v1/appointments

Create a new appointment (no auth required — client-facing).

**Auth:** None

**Request:**
```json
{
  "provider_slug": "barberia-juan",
  "employee_id": "emp-uuid-1",
  "service_id": "svc-uuid-1",
  "date": "2026-09-15",
  "start_time": "10:00",
  "client_name": "María García",
  "client_phone": "+521234567890"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| provider_slug | string | yes | existing active provider |
| employee_id | string | yes (or `"any"`) | existing active employee of provider. `"any"` = auto-assign. |
| service_id | string | yes | existing active service of provider |
| date | string | yes | `YYYY-MM-DD`, not in the past, within max advance days |
| start_time | string | yes | `HH:MM`, must be an available slot |
| client_name | string | yes | min 2, max 100 |
| client_phone | string | yes | valid phone format |

**Response (201 Created):**
```json
{
  "id": "apt-uuid-123",
  "provider": {
    "name": "Barbería Juan",
    "slug": "barberia-juan"
  },
  "employee": {
    "id": "emp-uuid-1",
    "name": "Juan Pérez"
  },
  "service": {
    "name": "Corte de pelo",
    "duration_minutes": 30
  },
  "date": "2026-09-15",
  "start_time": "10:00",
  "end_time": "10:30",
  "status": "confirmed",
  "whatsapp_link": "https://wa.me/521234567890?text=Hola%2C%20tengo%20un%20turno%20el%2015%2F09%20a%20las%2010%3A00"
}
```

**Errors:**
- `409 SLOT_TAKEN` — slot already booked (optimistic locking)
- `422 BUSINESS_RULE_VIOLATION` — max active appointments exceeded, or slot in the past

---

#### GET /v1/appointments/:id

View appointment details (for clients — no auth, but appointment UUID acts as a secret).

**Auth:** None

**Response (200 OK):**
```json
{
  "id": "apt-uuid-123",
  "provider": {
    "name": "Barbería Juan",
    "slug": "barberia-juan",
    "phone": "+521234567890"
  },
  "employee": {
    "id": "emp-uuid-1",
    "name": "Juan Pérez"
  },
  "service": {
    "name": "Corte de pelo"
  },
  "date": "2026-09-15",
  "start_time": "10:00",
  "end_time": "10:30",
  "status": "confirmed",
  "can_cancel": true,
  "can_reschedule": true,
  "whatsapp_link": "https://wa.me/521234567890?text=..."
}
```

---

#### PUT /v1/appointments/:id/cancel

Cancel an appointment (client-initiated).

**Auth:** None (verified by client_phone)

**Request:**
```json
{
  "client_phone": "+521234567890"
}
```

**Response (200 OK):**
```json
{
  "id": "apt-uuid-123",
  "status": "cancelled",
  "message": "Turno cancelado exitosamente."
}
```

**Errors:**
- `403 FORBIDDEN` — phone doesn't match
- `422 BUSINESS_RULE_VIOLATION` — too late to cancel (within minimum cancellation window)

---

#### PUT /v1/appointments/:id/reschedule

Reschedule an appointment (client-initiated). Atomic: releases old slot + books new.

**Auth:** None (verified by client_phone)

**Request:**
```json
{
  "client_phone": "+521234567890",
  "new_date": "2026-09-16",
  "new_start_time": "11:00"
}
```

**Response (200 OK):**
```json
{
  "id": "apt-uuid-123",
  "date": "2026-09-16",
  "start_time": "11:00",
  "end_time": "11:30",
  "status": "confirmed",
  "message": "Turno reagendado exitosamente."
}
```

**Errors:**
- `409 SLOT_TAKEN` — new slot already booked (original appointment preserved)
- `422 BUSINESS_RULE_VIOLATION` — too late to reschedule

---

### 8. Appointments — Provider (Authenticated)

#### GET /v1/appointments

List appointments for the provider/employee.

**Auth:** Bearer (provider or employee)

**Query params:**
- `date=2026-09-15` — single day
- `from=2026-09-15&to=2026-09-21` — date range
- `employee_id=emp-uuid-1` — filter by employee (admin only)
- `status=confirmed` — filter by status
- `page=1&per_page=50`

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "apt-uuid-123",
      "employee": {
        "id": "emp-uuid-1",
        "name": "Juan Pérez"
      },
      "service": {
        "name": "Corte de pelo"
      },
      "client_name": "María García",
      "client_phone": "+521234567890",
      "date": "2026-09-15",
      "start_time": "10:00",
      "end_time": "10:30",
      "status": "confirmed",
      "whatsapp_link": "https://wa.me/521234567890?text=..."
    }
  ],
  "total": 12,
  "page": 1,
  "per_page": 50
}
```

Employees only see their own appointments. Admins see all.

---

#### PUT /v1/appointments/:id/status

Update appointment status (provider-side).

**Auth:** Bearer (provider admin or owning employee)

**Request:**
```json
{
  "status": "completed"
}
```

| Status | Description | Billing Impact |
|--------|-------------|----------------|
| `completed` | Client attended, was served | +1 to monthly count |
| `no_show` | Client didn't show up | No billing impact |
| `cancelled` | Provider cancels | No billing impact |

**Response (200 OK):** Updated appointment object.

---

#### PUT /v1/appointments/:id

Edit appointment (provider-side — move to different time or employee).

**Auth:** Bearer (provider admin)

**Request:**
```json
{
  "employee_id": "emp-uuid-2",
  "date": "2026-09-16",
  "start_time": "14:00"
}
```

All fields optional. Atomic slot swap.

**Response (200 OK):** Updated appointment object.

---

#### POST /v1/appointments/manual

Create a manual/walk-in appointment (provider-side).

**Auth:** Bearer (provider or employee)

**Request:**
```json
{
  "employee_id": "emp-uuid-1",
  "service_id": "svc-uuid-1",
  "date": "2026-09-15",
  "start_time": "14:00",
  "client_name": "Walk-in Client",
  "client_phone": "+521234567899"
}
```

`client_phone` is optional for walk-ins.

**Response (201 Created):** Appointment object.

---

### 9. Search (Public)

#### GET /v1/search

Search providers by name or slug.

**Auth:** None

**Query params:**
- `q=barberia` — search term (min 2 chars)
- `page=1&per_page=20`

**Response (200 OK):**
```json
{
  "items": [
    {
      "slug": "barberia-juan",
      "name": "Barbería Juan",
      "type": "individual",
      "address": "Calle Falsa 123, CDMX",
      "services_count": 3,
      "rating": null
    }
  ],
  "total": 1,
  "page": 1,
  "per_page": 20
}
```

Search uses PostgreSQL `pg_trgm` for fuzzy matching on provider name + slug.

---

### 10. Push Tokens (Public)

#### POST /v1/push-tokens

Register an FCM token for push notifications.

**Auth:** None (or Bearer for providers/employees)

**Request:**
```json
{
  "token": "fcm-token-abc123...",
  "platform": "ios",
  "client_phone": "+521234567890"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| token | string | yes | FCM device token |
| platform | enum | yes | `"ios"`, `"android"`, `"web"` |
| client_phone | string | no | For unregistered clients. Omit if authenticated provider/employee. |

**Response (201 Created):**
```json
{
  "id": "pt-uuid-1",
  "registered": true
}
```

---

#### DELETE /v1/push-tokens/:token

Deregister an FCM token.

**Auth:** None

**Response (204 No Content)**

---

### 11. Employee Join (Public)

#### POST /v1/join

Accept an invitation code to join a business.

**Auth:** None (creates employee account)

**Request:**
```json
{
  "code": "ABC123XY",
  "phone": "+521234567891",
  "email": "maria@example.com",
  "password": "securepassword123"
}
```

**Response (200 OK):**
```json
{
  "employee_id": "emp-uuid-1",
  "provider": {
    "name": "Barbería Juan",
    "slug": "barberia-juan"
  },
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh..."
}
```

**Errors:**
- `404 NOT_FOUND` — invalid code
- `422 BUSINESS_RULE_VIOLATION` — code expired or already used

---

### 12. Provider Settings (Authenticated)

#### GET /v1/provider/me/settings

**Auth:** Bearer (provider admin)

**Response (200 OK):**
```json
{
  "min_cancellation_hours": 2,
  "max_advance_days": 30,
  "max_active_appointments_per_client": 2,
  "allow_reschedule": true,
  "business_hours": {
    "start_time": "08:00",
    "end_time": "20:00"
  },
  "notification_preferences": {
    "weekly_summary": true,
    "daily_reminder": true,
    "quiet_hours_start": "21:00",
    "quiet_hours_end": "08:00"
  }
}
```

---

#### PUT /v1/provider/me/settings

**Auth:** Bearer (provider admin)

**Request:** Same shape as GET response (partial update).

**Response (200 OK):** Updated settings object.

---

### 13. Health (Internal)

#### GET /health

**Auth:** None

**Response (200 OK):**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 86400
}
```

Used by UptimeRobot, load balancers, and Prometheus.

---

## JWT Token Structure

**Access Token Payload:**
```json
{
  "sub": "prov-uuid-123",
  "role": "admin",
  "provider_id": "prov-uuid-123",
  "type": "access",
  "iat": 1726387200,
  "exp": 1726388100
}
```

| Claim | Description |
|-------|-------------|
| sub | User ID (provider_id or employee_id) |
| role | `"admin"` (provider/PJ admin) or `"employee"` |
| provider_id | Always present — the provider this user belongs to |
| type | `"access"` (15min) or `"refresh"` (7 days) |

**Employee Access Token:**
```json
{
  "sub": "emp-uuid-1",
  "role": "employee",
  "provider_id": "prov-uuid-123",
  "type": "access",
  "iat": 1726387200,
  "exp": 1726388100
}
```
