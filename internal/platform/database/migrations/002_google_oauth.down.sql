ALTER TABLE employees DROP COLUMN IF EXISTS google_id;

-- Restore NOT NULL on password_hash (set empty hash for any Google-only accounts)
UPDATE providers SET password_hash = '' WHERE password_hash IS NULL;
ALTER TABLE providers ALTER COLUMN password_hash SET NOT NULL;

ALTER TABLE providers DROP COLUMN IF EXISTS google_id;
