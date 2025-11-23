-- +goose Up
CREATE TABLE IF NOT EXISTS pr_reviewers (
                                            pr_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
                                            reviewer_id TEXT NOT NULL REFERENCES users(id),
                                            PRIMARY KEY (pr_id, reviewer_id)
);

-- +goose Down
DROP TABLE IF EXISTS pr_reviewers;