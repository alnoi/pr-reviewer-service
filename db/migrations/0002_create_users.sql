-- +goose Up
CREATE TABLE IF NOT EXISTS users (
                                     id TEXT PRIMARY KEY,
                                     team_name TEXT NOT NULL REFERENCES teams(team_name) ON DELETE CASCADE,
                                     username TEXT NOT NULL,
                                     is_active BOOLEAN NOT NULL DEFAULT TRUE,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_users_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_update_users_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
EXECUTE FUNCTION update_users_timestamp();

-- +goose Down
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_users_timestamp();