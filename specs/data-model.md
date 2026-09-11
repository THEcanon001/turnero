# Data Model: Turnero App

---

## Schema Overview

```
providers ─────┬──── employees ────── schedules
               │         │            schedule_exceptions
               │         │
               │         ├──── appointments (employee_id + provider_id)
               │         │
               │         └──── push_tokens (employee_id)
               │
               ├──── push_tokens (provider_id)
               ├──── billing_usage ──── billing_transactions
               ├──── refresh_tokens (provider_id)
               └──── audit_log

push_tokens (client_phone) ── standalone for unregistered clients
```

---

## Complete SQL Schema

```sql
-- ============================================================
-- EXTENSIONS
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- Fuzzy search on provider names

-- ============================================================
-- ENUM TYPES
-- ============================================================

CREATE TYPE provider_type AS ENUM ('business', 'individual');
CREATE TYPE employee_role AS ENUM ('admin', 'employee');
CREATE TYPE appointment_status AS ENUM ('confirmed', 'cancelled', 'completed', 'no_show');
CREATE TYPE billing_plan AS ENUM ('free', 'basic', 'pro');

-- ============================================================
-- TABLE: providers
-- Proveedores de servicio (PF individual o PJ negocio)
-- ============================================================

CREATE TABLE providers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type              provider_type NOT NULL,

    -- Common data
    name              TEXT NOT NULL,
    slug              TEXT UNIQUE NOT NULL,          -- Unique handle for URLs and search (e.g., "barberia-juan")
    phone             TEXT NOT NULL,                 -- WhatsApp contact number
    address           TEXT,
    timezone          TEXT NOT NULL DEFAULT 'America/Mexico_City',

    -- Business-only fields (type = 'business')
    business_name     TEXT,                          -- Legal name

    -- Auth
    email             TEXT UNIQUE NOT NULL,
    password_hash     TEXT NOT NULL,

    -- QR
    qr_image_path     TEXT,                          -- Path to generated PNG

    -- Billing
    plan              billing_plan NOT NULL DEFAULT 'free',
    billing_cycle_start DATE,                        -- Start of current billing cycle

    -- Metadata
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Relationships: One provider has many employees, many appointments, many billing records.
-- Constraints:
--   slug: unique, case-insensitive (enforced at application level), only letters/numbers/hyphens
--   email: unique

-- ============================================================
-- TABLE: employees
-- Empleados del proveedor (including the provider themselves for PF)
-- For PF providers, a single "self" employee record is auto-created.
-- ============================================================

CREATE TABLE employees (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    name              TEXT NOT NULL,
    phone             TEXT NOT NULL,
    role              employee_role NOT NULL DEFAULT 'employee',

    -- Auth (optional — only if employee needs own login)
    email             TEXT UNIQUE,
    password_hash     TEXT,

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Relationships: Belongs to one provider. Has many schedules, appointments.
-- For PJ: admin invites employees via invitation codes.

-- ============================================================
-- TABLE: services
-- Servicios ofrecidos por un proveedor
-- ============================================================

CREATE TABLE services (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    name              TEXT NOT NULL,
    description       TEXT,
    duration_minutes  SMALLINT NOT NULL CHECK (duration_minutes > 0),

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Relationships: Belongs to one provider. Many-to-many with employees (via employee_services).

-- ============================================================
-- TABLE: employee_services
-- Qué empleados pueden realizar qué servicios (N:M)
-- ============================================================

CREATE TABLE employee_services (
    employee_id       UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    service_id        UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,

    PRIMARY KEY (employee_id, service_id)
);

-- For PF providers, the single employee is auto-assigned all services.

-- ============================================================
-- TABLE: schedules
-- Horarios de atención semanales por empleado
-- ============================================================

CREATE TABLE schedules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id       UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,

    day_of_week       SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),  -- 0=Sunday, 6=Saturday
    start_time        TIME NOT NULL,
    end_time          TIME NOT NULL,
    slot_duration_minutes SMALLINT NOT NULL DEFAULT 30 CHECK (slot_duration_minutes > 0),
    break_after_slot_minutes SMALLINT NOT NULL DEFAULT 0,  -- Rest between appointments

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One schedule per employee per day
    UNIQUE (employee_id, day_of_week),
    -- Validate start < end
    CHECK (start_time < end_time)
);

-- Relationships: Belongs to one employee.
-- Business rule: For PJ, employee schedule must fall within provider's business hours.

-- ============================================================
-- TABLE: schedule_exceptions
-- Overrides for specific dates (holidays, vacations, special hours)
-- ============================================================

CREATE TABLE schedule_exceptions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id       UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,

    date              DATE NOT NULL,
    is_available      BOOLEAN NOT NULL DEFAULT false,  -- false = day off, true = special hours
    start_time        TIME,                            -- Only if is_available = true
    end_time          TIME,                            -- Only if is_available = true
    reason            TEXT,                            -- "Holiday", "Vacation", etc.

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (employee_id, date)
);

-- ============================================================
-- TABLE: appointments
-- Turnos/citas agendadas
-- ============================================================

CREATE TABLE appointments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id),
    employee_id       UUID NOT NULL REFERENCES employees(id),
    service_id        UUID REFERENCES services(id),

    -- Client data (no registration required)
    client_name       TEXT NOT NULL,
    client_phone      TEXT NOT NULL,

    -- Appointment slot
    date              DATE NOT NULL,
    start_time        TIME NOT NULL,
    end_time          TIME NOT NULL,
    status            appointment_status NOT NULL DEFAULT 'confirmed',

    notes             TEXT,
    cancellation_reason TEXT,

    -- Tracking
    reminder_sent     BOOLEAN NOT NULL DEFAULT false,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Prevent double booking: one employee cannot have two appointments at the same time
    -- (only for active appointments, not cancelled)
    UNIQUE (employee_id, date, start_time)
);

-- Relationships: Belongs to one provider + one employee. Optionally linked to one service.
-- Business rules:
--   - Cancelled appointments: the UNIQUE constraint remains (slot is still "used").
--     To free the slot on cancellation, delete + re-insert or use a partial unique index.
--   - Concurrency: optimistic locking — first INSERT wins, second gets unique violation.

-- ============================================================
-- TABLE: invitation_codes
-- Códigos de invitación para empleados
-- ============================================================

CREATE TABLE invitation_codes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    code              TEXT NOT NULL UNIQUE,
    employee_name     TEXT NOT NULL,                  -- Pre-filled name for the invited employee

    expires_at        TIMESTAMPTZ NOT NULL,           -- Default: 48h from creation
    used_at           TIMESTAMPTZ,                    -- NULL if not yet used
    used_by           UUID REFERENCES employees(id),  -- Employee who used the code

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: billing_usage
-- Monthly usage tracking for billing
-- ============================================================

CREATE TABLE billing_usage (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    month             DATE NOT NULL,                  -- First day of month (e.g., 2026-09-01)
    completed_appointments INT NOT NULL DEFAULT 0,    -- Completed appointments in the month
    free_tier_limit   INT NOT NULL DEFAULT 20,        -- Free appointments per month
    billable_appointments INT NOT NULL DEFAULT 0,     -- Appointments above free tier
    amount_due        DECIMAL(10,2) NOT NULL DEFAULT 0,  -- Amount to charge
    is_paid           BOOLEAN NOT NULL DEFAULT false,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (provider_id, month)
);

-- Business rules:
--   - PJ: ALL employees' completed appointments are summed.
--   - Cap: amount_due never exceeds ~$10 USD equivalent.
--   - Billing is post-use: charged at beginning of next month.

-- ============================================================
-- TABLE: billing_transactions
-- Payment history
-- ============================================================

CREATE TABLE billing_transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id),
    billing_usage_id  UUID REFERENCES billing_usage(id),

    amount            DECIMAL(10,2) NOT NULL,
    currency          TEXT NOT NULL DEFAULT 'MXN',
    description       TEXT,

    -- Payment gateway integration
    external_payment_id TEXT,                         -- Stripe/MercadoPago ID

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: push_tokens
-- FCM tokens for push notifications
-- ============================================================

CREATE TABLE push_tokens (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Can belong to provider/employee (with login) or client (by phone)
    provider_id       UUID REFERENCES providers(id) ON DELETE CASCADE,
    employee_id       UUID REFERENCES employees(id) ON DELETE CASCADE,
    client_phone      TEXT,                           -- For unregistered clients

    token             TEXT NOT NULL,
    platform          TEXT NOT NULL CHECK (platform IN ('ios', 'android', 'web')),

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: refresh_tokens
-- JWT refresh token rotation
-- ============================================================

CREATE TABLE refresh_tokens (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Can be provider or employee
    provider_id       UUID REFERENCES providers(id) ON DELETE CASCADE,
    employee_id       UUID REFERENCES employees(id) ON DELETE CASCADE,

    token_hash        TEXT NOT NULL UNIQUE,            -- SHA-256 of refresh token
    expires_at        TIMESTAMPTZ NOT NULL,
    revoked_at        TIMESTAMPTZ,
    replaced_by       UUID REFERENCES refresh_tokens(id),  -- Token rotation chain

    user_agent        TEXT,
    ip_address        INET,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (
        (provider_id IS NOT NULL AND employee_id IS NULL) OR
        (provider_id IS NULL AND employee_id IS NOT NULL)
    )
);

-- ============================================================
-- TABLE: audit_log
-- Action log for debugging and security
-- ============================================================

CREATE TABLE audit_log (
    id                BIGSERIAL PRIMARY KEY,

    actor_type        TEXT NOT NULL CHECK (actor_type IN ('provider', 'employee', 'client', 'system')),
    actor_id          TEXT,                            -- UUID of actor or 'system'

    action            TEXT NOT NULL,                   -- 'appointment.created', 'provider.updated', etc.
    resource_type     TEXT NOT NULL,                   -- 'appointment', 'provider', 'employee'
    resource_id       UUID,

    details           JSONB,                           -- Change payload
    ip_address        INET,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- INDEXES
-- ============================================================

-- Providers
CREATE INDEX idx_providers_slug ON providers (slug);
CREATE INDEX idx_providers_email ON providers (email);
CREATE INDEX idx_providers_name_trgm ON providers USING gin (name gin_trgm_ops);  -- Fuzzy search

-- Employees
CREATE INDEX idx_employees_provider ON employees (provider_id);

-- Services
CREATE INDEX idx_services_provider ON services (provider_id) WHERE is_active = true;

-- Schedules
CREATE INDEX idx_schedules_employee ON schedules (employee_id);

-- Schedule exceptions
CREATE INDEX idx_schedule_exceptions_lookup ON schedule_exceptions (employee_id, date);

-- Appointments
CREATE INDEX idx_appointments_employee_date ON appointments (employee_id, date);
CREATE INDEX idx_appointments_provider_date ON appointments (provider_id, date);
CREATE INDEX idx_appointments_status ON appointments (status) WHERE status = 'confirmed';
CREATE INDEX idx_appointments_reminder ON appointments (date, reminder_sent)
    WHERE status = 'confirmed' AND reminder_sent = false;
CREATE INDEX idx_appointments_client_phone ON appointments (client_phone, provider_id);

-- Billing
CREATE INDEX idx_billing_provider_month ON billing_usage (provider_id, month);

-- Push tokens
CREATE INDEX idx_push_tokens_provider ON push_tokens (provider_id) WHERE is_active = true;
CREATE INDEX idx_push_tokens_employee ON push_tokens (employee_id) WHERE is_active = true;
CREATE INDEX idx_push_tokens_phone ON push_tokens (client_phone) WHERE is_active = true;

-- Refresh tokens
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens (token_hash) WHERE revoked_at IS NULL;

-- Invitation codes
CREATE INDEX idx_invitation_codes_code ON invitation_codes (code) WHERE used_at IS NULL;
CREATE INDEX idx_invitation_codes_provider ON invitation_codes (provider_id);

-- Audit log
CREATE INDEX idx_audit_log_actor ON audit_log (actor_type, actor_id);
CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id);
CREATE INDEX idx_audit_log_created ON audit_log (created_at);

-- ============================================================
-- TRIGGERS
-- ============================================================

-- Auto-update updated_at on row modification
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_providers_updated_at
    BEFORE UPDATE ON providers FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_employees_updated_at
    BEFORE UPDATE ON employees FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_services_updated_at
    BEFORE UPDATE ON services FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_appointments_updated_at
    BEFORE UPDATE ON appointments FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_billing_usage_updated_at
    BEFORE UPDATE ON billing_usage FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_push_tokens_updated_at
    BEFORE UPDATE ON push_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

---

## Entity Relationships

```
providers (1) ──── (N) employees
providers (1) ──── (N) services
employees (N) ──── (M) services       [via employee_services]
employees (1) ──── (N) schedules      [max 1 per day_of_week]
employees (1) ──── (N) schedule_exceptions
employees (1) ──── (N) appointments
providers (1) ──── (N) appointments
services  (1) ──── (N) appointments   [optional FK]
providers (1) ──── (N) billing_usage
billing_usage (1) ── (N) billing_transactions
providers (1) ──── (N) invitation_codes
providers (1) ──── (N) push_tokens
employees (1) ──── (N) push_tokens
providers (1) ──── (N) refresh_tokens
employees (1) ──── (N) refresh_tokens
```

---

## Access Policies

| Table | Actor | Access |
|-------|-------|--------|
| providers | Provider (owner) | Full CRUD on own record |
| providers | Client | Read public fields (name, slug, phone, address) |
| employees | Admin PJ | Full CRUD on all employees of their provider |
| employees | Employee | Read own record, update own schedule |
| services | Admin PJ / PF | Full CRUD |
| services | Client | Read (active only) |
| schedules | Admin PJ | CRUD any employee's schedule |
| schedules | Employee | CRUD own schedule (within business hours) |
| schedules | Client | Read (via availability endpoint — computed slots, not raw schedules) |
| appointments | Client | Create (public). Read/cancel own (verified by client_phone). |
| appointments | Provider PF | Full CRUD on all own appointments |
| appointments | Employee PJ | CRUD own appointments only |
| appointments | Admin PJ | Full CRUD on all appointments of all employees |
| billing_usage | Provider / Admin PJ | Read only |
| audit_log | System | Write only. Not exposed via API. |

---

## Data Retention

| Data | Retention Policy |
|------|-----------------|
| Client data (name, phone) in appointments | Auto-delete after 12 months of inactivity (no new appointments) |
| Completed/cancelled appointments | Archive to cold storage after 24 months |
| Audit log | Retain 12 months, then purge |
| Push tokens | Remove inactive tokens after 90 days |
| Refresh tokens | Expired tokens purged daily by cron |
| Billing data | Retain indefinitely (financial records) |
