-- +goose Up

CREATE INDEX IF NOT EXISTS idx_users_team_active
    ON users(team_name, is_active);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer
    ON pr_reviewers(reviewer_id);

-- +goose Down

DROP INDEX IF EXISTS idx_users_team_active;
DROP INDEX IF EXISTS idx_pr_reviewers_reviewer;
