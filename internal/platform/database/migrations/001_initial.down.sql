-- Drop triggers first
DROP TRIGGER IF EXISTS trg_push_tokens_updated_at ON push_tokens;
DROP TRIGGER IF EXISTS trg_billing_usage_updated_at ON billing_usage;
DROP TRIGGER IF EXISTS trg_appointments_updated_at ON appointments;
DROP TRIGGER IF EXISTS trg_services_updated_at ON services;
DROP TRIGGER IF EXISTS trg_employees_updated_at ON employees;
DROP TRIGGER IF EXISTS trg_providers_updated_at ON providers;

DROP FUNCTION IF EXISTS update_updated_at();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS push_tokens;
DROP TABLE IF EXISTS billing_transactions;
DROP TABLE IF EXISTS billing_usage;
DROP TABLE IF EXISTS invitation_codes;
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS schedule_exceptions;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS employee_services;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS providers;

-- Drop enum types
DROP TYPE IF EXISTS billing_plan;
DROP TYPE IF EXISTS appointment_status;
DROP TYPE IF EXISTS employee_role;
DROP TYPE IF EXISTS provider_type;

-- Drop extensions
DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "pgcrypto";
