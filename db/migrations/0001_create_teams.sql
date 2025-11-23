-- +goose Up
CREATE TABLE IF NOT EXISTS teams (
                                     team_name TEXT PRIMARY KEY,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_teams_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- триггер
CREATE TRIGGER trigger_update_teams_timestamp
    BEFORE UPDATE ON teams
    FOR EACH ROW
EXECUTE FUNCTION update_teams_timestamp();

-- +goose Down
DROP TABLE IF EXISTS teams;
DROP FUNCTION IF EXISTS update_teams_timestamp();