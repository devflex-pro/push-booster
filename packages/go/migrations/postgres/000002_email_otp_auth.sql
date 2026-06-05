-- +goose Up
INSERT INTO roles (id, code, name)
VALUES
    ('00000000-0000-4000-8000-000000000001', 'admin', 'Admin'),
    ('00000000-0000-4000-8000-000000000002', 'user', 'User')
ON CONFLICT (code) DO NOTHING;

ALTER TABLE users
    ALTER COLUMN password_hash SET DEFAULT '',
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN approved BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN otp_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN otp_expires_at TIMESTAMPTZ,
    ADD COLUMN otp_requested_at TIMESTAMPTZ;

CREATE INDEX users_otp_expires_at_idx ON users(otp_expires_at);

-- +goose Down
DROP INDEX users_otp_expires_at_idx;

ALTER TABLE users
    DROP COLUMN otp_requested_at,
    DROP COLUMN otp_expires_at,
    DROP COLUMN otp_hash,
    DROP COLUMN approved,
    DROP COLUMN email_verified,
    ALTER COLUMN password_hash DROP DEFAULT;
