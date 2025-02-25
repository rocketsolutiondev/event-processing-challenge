BEGIN;

CREATE TABLE IF NOT EXISTS players (
    id bigserial PRIMARY KEY,
    email text NOT NULL,
    last_signed_in_at timestamptz
);

INSERT INTO players (id, email, last_signed_in_at) VALUES
    (10, 'john@example.com', now() - interval '1h'),
    (11, 'jane@example.com', now() - interval '3h'),
    (12, 'bob@example.com', now() - interval '2d'),
    (13, 'rick@example.com', now() - interval '5h'),
    (14, 'morty@example.com', now() - interval '1d')
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    last_signed_in_at = EXCLUDED.last_signed_in_at;

COMMIT;
