-- +goose Up
CREATE TABLE IF NOT EXISTS pull_requests (
                                             id TEXT PRIMARY KEY,
                                             pull_request_name TEXT NOT NULL,
                                             author_id TEXT NOT NULL REFERENCES users(id),
                                             status TEXT NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
                                             created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                             updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                             merged_at TIMESTAMPTZ
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_pull_requests_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_update_pull_requests_timestamp
    BEFORE UPDATE ON pull_requests
    FOR EACH ROW
EXECUTE FUNCTION update_pull_requests_timestamp();

-- +goose Down
DROP TABLE IF EXISTS pull_requests;
DROP FUNCTION IF EXISTS update_pull_requests_timestamp();