package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

type PRRepository struct {
	pool *pgxpool.Pool
}

func NewPRRepository(pool *pgxpool.Pool) *PRRepository {
	return &PRRepository{pool: pool}
}

func (r *PRRepository) CreatePR(ctx context.Context, pr domain.PullRequest) error {
	const query = `
		INSERT INTO pull_requests (
			id,
			pull_request_name,
			author_id,
			status
		)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.AuthorID,
		string(pr.Status),
	)
	return err
}

func (r *PRRepository) PRExists(ctx context.Context, prID string) (bool, error) {
	const q = `SELECT 1 FROM pull_requests WHERE id = $1`

	var x int
	err := r.pool.QueryRow(ctx, q, prID).Scan(&x)

	if err == nil {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (r *PRRepository) GetPR(ctx context.Context, prID string) (domain.PullRequest, error) {
	var res domain.PullRequest
	const q = `
		SELECT id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE id = $1
	`

	var (
		id        string
		name      string
		authorID  string
		status    string
		createdAt time.Time
		mergedAt  *time.Time
	)

	err := r.pool.QueryRow(ctx, q, prID).Scan(
		&id, &name, &authorID, &status, &createdAt, &mergedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, domain.NewDomainError(domain.ErrorCodeNotFound, "PR not found")
		}
		return res, err
	}

	reviewers, err := r.GetPRReviewers(ctx, prID)
	if err != nil {
		return res, err
	}

	res = domain.PullRequest{
		PullRequestID:     id,
		PullRequestName:   name,
		AuthorID:          authorID,
		Status:            domain.PRStatus(status),
		AssignedReviewers: reviewers,
		CreatedAt:         createdAt,
		MergedAt:          mergedAt,
	}

	return res, nil
}

func (r *PRRepository) UpdatePR(ctx context.Context, pr domain.PullRequest) error {
	const q = `
		UPDATE pull_requests
		SET pull_request_name = $2,
		    author_id = $3,
		    status = $4,
		    merged_at = $5
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.AuthorID,
		string(pr.Status),
		pr.MergedAt,
	)
	return err
}

func (r *PRRepository) GetPRReviewers(ctx context.Context, prID string) ([]string, error) {
	const q = `
		SELECT reviewer_id
		FROM pr_reviewers
		WHERE pr_id = $1
		ORDER BY reviewer_id
	`

	rows, err := r.pool.Query(ctx, q, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, userID)
	}

	return reviewers, rows.Err()
}

func (r *PRRepository) SetPRReviewers(ctx context.Context, prID string, reviewers []string) error {
	const deleteQ = `DELETE FROM pr_reviewers WHERE pr_id = $1`
	if _, err := r.pool.Exec(ctx, deleteQ, prID); err != nil {
		return err
	}

	const insertQ = `
        INSERT INTO pr_reviewers (pr_id, reviewer_id)
        SELECT $1, unnest($2::text[])
    `

	_, err := r.pool.Exec(ctx, insertQ, prID, reviewers)
	return err
}

func (r *PRRepository) GetPRsWhereReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	const q = `
		SELECT pr.id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN pr_reviewers r ON pr.id = r.pr_id
		WHERE r.reviewer_id = $1
		ORDER BY pr.created_at
	`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []domain.PullRequestShort

	for rows.Next() {
		var (
			id        string
			name      string
			author    string
			status    string
			createdAt time.Time
			mergedAt  *time.Time
		)

		if err := rows.Scan(&id, &name, &author, &status, &createdAt, &mergedAt); err != nil {
			return nil, err
		}

		item := domain.PullRequestShort{
			PullRequestID:   id,
			PullRequestName: name,
			AuthorID:        author,
			Status:          domain.PRStatus(status),
		}

		prs = append(prs, item)
	}

	return prs, rows.Err()
}

func (r *PRRepository) GetOpenPRsByReviewers(ctx context.Context, userIDs []string) ([]domain.PullRequest, error) {
	if len(userIDs) == 0 {
		return []domain.PullRequest{}, nil
	}

	const q = `
		SELECT DISTINCT pr.id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN pr_reviewers r ON pr.id = r.pr_id
		WHERE r.reviewer_id = ANY($1)
		  AND pr.status = 'OPEN'
	`

	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []domain.PullRequest

	for rows.Next() {
		var (
			id        string
			name      string
			author    string
			status    string
			createdAt time.Time
			mergedAt  *time.Time
		)

		if err := rows.Scan(&id, &name, &author, &status, &createdAt, &mergedAt); err != nil {
			return nil, err
		}

		reviewers, err := r.GetPRReviewers(ctx, id)
		if err != nil {
			return nil, err
		}

		pr := domain.PullRequest{
			PullRequestID:     id,
			PullRequestName:   name,
			AuthorID:          author,
			Status:            domain.PRStatus(status),
			AssignedReviewers: reviewers,
			CreatedAt:         createdAt,
			MergedAt:          mergedAt,
		}

		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

func (r *PRRepository) GetAssignmentsCountByUser(ctx context.Context) ([]domain.UserAssignmentsStat, error) {
	const q = `
		SELECT u.id, COALESCE(COUNT(r.pr_id), 0) AS assignments_count
		FROM users u
		LEFT JOIN pr_reviewers r ON u.id = r.reviewer_id
		GROUP BY u.id
		ORDER BY assignments_count DESC
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.UserAssignmentsStat

	for rows.Next() {
		var userID string
		var count int

		if err := rows.Scan(&userID, &count); err != nil {
			return nil, err
		}

		stats = append(stats, domain.UserAssignmentsStat{
			UserID:                 userID,
			ReviewAssignmentsCount: count,
		})
	}

	return stats, rows.Err()
}

func (r *PRRepository) GetPRStatusCounts(ctx context.Context) (domain.PRStatusCounts, error) {
	const q = `
		SELECT
			COUNT(*) FILTER (WHERE status = 'OPEN')   AS open_count,
			COUNT(*) FILTER (WHERE status = 'MERGED') AS merged_count,
			COUNT(*)                                  AS total_count
		FROM pull_requests
	`

	var res domain.PRStatusCounts
	if err := r.pool.QueryRow(ctx, q).Scan(&res.Open, &res.Merged, &res.Total); err != nil {
		return domain.PRStatusCounts{}, err
	}

	return res, nil
}
