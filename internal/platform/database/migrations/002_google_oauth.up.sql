-- Add Google OAuth support: google_id column on providers and employees.
-- password_hash becomes nullable for Google-only accounts.

ALTER TABLE providers ADD COLUMN google_id TEXT UNIQUE;
ALTER TABLE providers ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE employees ADD COLUMN google_id TEXT UNIQUE;
