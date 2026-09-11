-- ============================================================
-- EXTENSIONS
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- ENUM TYPES
-- ============================================================

CREATE TYPE provider_type AS ENUM ('business', 'individual');
CREATE TYPE employee_role AS ENUM ('admin', 'employee');
CREATE TYPE appointment_status AS ENUM ('confirmed', 'cancelled', 'completed', 'no_show');
CREATE TYPE billing_plan AS ENUM ('free', 'basic', 'pro');

-- ============================================================
-- TABLE: providers
-- ============================================================

CREATE TABLE providers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type              provider_type NOT NULL,

    name              TEXT NOT NULL,
    slug              TEXT UNIQUE NOT NULL,
    phone             TEXT NOT NULL,
    address           TEXT,
    timezone          TEXT NOT NULL DEFAULT 'America/Mexico_City',

    business_name     TEXT,

    email             TEXT UNIQUE NOT NULL,
    password_hash     TEXT NOT NULL,

    qr_image_path     TEXT,

    plan              billing_plan NOT NULL DEFAULT 'free',
    billing_cycle_start DATE,

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: employees
-- ============================================================

CREATE TABLE employees (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    name              TEXT NOT NULL,
    phone             TEXT NOT NULL,
    role              employee_role NOT NULL DEFAULT 'employee',

    email             TEXT UNIQUE,
    password_hash     TEXT,

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: services
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

-- ============================================================
-- TABLE: employee_services
-- ============================================================

CREATE TABLE employee_services (
    employee_id       UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    service_id        UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,

    PRIMARY KEY (employee_id, service_id)
);

-- ============================================================
-- TABLE: schedules
-- ============================================================

CREATE TABLE schedules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id       UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,

    day_of_week       SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time        TIME NOT NULL,
    end_time          TIME NOT NULL,
    slot_duration_minutes SMALLINT NOT NULL DEFAULT 30 CHECK (slot_duration_minutes > 0),
    break_after_slot_minutes SMALLINT NOT NULL DEFAULT 0,

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (employee_id, day_of_week),
    CHECK (start_time < end_time)
);

-- ============================================================
-- TABLE: schedule_exceptions
-- ============================================================

CREATE TABLE schedule_exceptions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id       UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,

    date              DATE NOT NULL,
    is_available      BOOLEAN NOT NULL DEFAULT false,
    start_time        TIME,
    end_time          TIME,
    reason            TEXT,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (employee_id, date)
);

-- ============================================================
-- TABLE: appointments
-- ============================================================

CREATE TABLE appointments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id),
    employee_id       UUID NOT NULL REFERENCES employees(id),
    service_id        UUID REFERENCES services(id),

    client_name       TEXT NOT NULL,
    client_phone      TEXT NOT NULL,

    date              DATE NOT NULL,
    start_time        TIME NOT NULL,
    end_time          TIME NOT NULL,
    status            appointment_status NOT NULL DEFAULT 'confirmed',

    notes             TEXT,
    cancellation_reason TEXT,

    reminder_sent     BOOLEAN NOT NULL DEFAULT false,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (employee_id, date, start_time)
);

-- ============================================================
-- TABLE: invitation_codes
-- ============================================================

CREATE TABLE invitation_codes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    code              TEXT NOT NULL UNIQUE,
    employee_name     TEXT NOT NULL,

    expires_at        TIMESTAMPTZ NOT NULL,
    used_at           TIMESTAMPTZ,
    used_by           UUID REFERENCES employees(id),

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: billing_usage
-- ============================================================

CREATE TABLE billing_usage (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,

    month             DATE NOT NULL,
    completed_appointments INT NOT NULL DEFAULT 0,
    free_tier_limit   INT NOT NULL DEFAULT 20,
    billable_appointments INT NOT NULL DEFAULT 0,
    amount_due        DECIMAL(10,2) NOT NULL DEFAULT 0,
    is_paid           BOOLEAN NOT NULL DEFAULT false,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (provider_id, month)
);

-- ============================================================
-- TABLE: billing_transactions
-- ============================================================

CREATE TABLE billing_transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id       UUID NOT NULL REFERENCES providers(id),
    billing_usage_id  UUID REFERENCES billing_usage(id),

    amount            DECIMAL(10,2) NOT NULL,
    currency          TEXT NOT NULL DEFAULT 'MXN',
    description       TEXT,

    external_payment_id TEXT,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: push_tokens
-- ============================================================

CREATE TABLE push_tokens (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    provider_id       UUID REFERENCES providers(id) ON DELETE CASCADE,
    employee_id       UUID REFERENCES employees(id) ON DELETE CASCADE,
    client_phone      TEXT,

    token             TEXT NOT NULL,
    platform          TEXT NOT NULL CHECK (platform IN ('ios', 'android', 'web')),

    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TABLE: refresh_tokens
-- ============================================================

CREATE TABLE refresh_tokens (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    provider_id       UUID REFERENCES providers(id) ON DELETE CASCADE,
    employee_id       UUID REFERENCES employees(id) ON DELETE CASCADE,

    token_hash        TEXT NOT NULL UNIQUE,
    expires_at        TIMESTAMPTZ NOT NULL,
    revoked_at        TIMESTAMPTZ,
    replaced_by       UUID REFERENCES refresh_tokens(id),

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
-- ============================================================

CREATE TABLE audit_log (
    id                BIGSERIAL PRIMARY KEY,

    actor_type        TEXT NOT NULL CHECK (actor_type IN ('provider', 'employee', 'client', 'system')),
    actor_id          TEXT,

    action            TEXT NOT NULL,
    resource_type     TEXT NOT NULL,
    resource_id       UUID,

    details           JSONB,
    ip_address        INET,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- INDEXES
-- ============================================================

CREATE INDEX idx_providers_slug ON providers (slug);
CREATE INDEX idx_providers_email ON providers (email);
CREATE INDEX idx_providers_name_trgm ON providers USING gin (name gin_trgm_ops);

CREATE INDEX idx_employees_provider ON employees (provider_id);

CREATE INDEX idx_services_provider ON services (provider_id) WHERE is_active = true;

CREATE INDEX idx_schedules_employee ON schedules (employee_id);

CREATE INDEX idx_schedule_exceptions_lookup ON schedule_exceptions (employee_id, date);

CREATE INDEX idx_appointments_employee_date ON appointments (employee_id, date);
CREATE INDEX idx_appointments_provider_date ON appointments (provider_id, date);
CREATE INDEX idx_appointments_status ON appointments (status) WHERE status = 'confirmed';
CREATE INDEX idx_appointments_reminder ON appointments (date, reminder_sent)
    WHERE status = 'confirmed' AND reminder_sent = false;
CREATE INDEX idx_appointments_client_phone ON appointments (client_phone, provider_id);

CREATE INDEX idx_billing_provider_month ON billing_usage (provider_id, month);

CREATE INDEX idx_push_tokens_provider ON push_tokens (provider_id) WHERE is_active = true;
CREATE INDEX idx_push_tokens_employee ON push_tokens (employee_id) WHERE is_active = true;
CREATE INDEX idx_push_tokens_phone ON push_tokens (client_phone) WHERE is_active = true;

CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens (token_hash) WHERE revoked_at IS NULL;

CREATE INDEX idx_invitation_codes_code ON invitation_codes (code) WHERE used_at IS NULL;
CREATE INDEX idx_invitation_codes_provider ON invitation_codes (provider_id);

CREATE INDEX idx_audit_log_actor ON audit_log (actor_type, actor_id);
CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id);
CREATE INDEX idx_audit_log_created ON audit_log (created_at);

-- ============================================================
-- TRIGGERS
-- ============================================================

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
