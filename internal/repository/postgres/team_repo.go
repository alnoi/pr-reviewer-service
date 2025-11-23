package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

type TeamRepository struct {
	pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{
		pool: pool,
	}
}

func (r *TeamRepository) CreateTeam(ctx context.Context, teamName string) error {
	const query = `
		INSERT INTO teams (team_name)
		VALUES ($1)
	`
	_, err := r.pool.Exec(ctx, query, teamName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.NewDomainError(domain.ErrorCodeTeamExists, "team already exists")
		}
		return err
	}

	return nil
}

func (r *TeamRepository) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	const q = `
		SELECT u.id, u.username, u.is_active
		FROM teams t
		LEFT JOIN users u ON u.team_name = t.team_name
		WHERE t.team_name = $1
		ORDER BY u.username
	`

	var res domain.Team

	rows, err := r.pool.Query(ctx, q, teamName)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	var (
		foundTeam bool
		members   []domain.TeamMember
	)

	for rows.Next() {
		foundTeam = true

		var (
			userID   *string
			username *string
			isActive *bool
		)

		if err := rows.Scan(&userID, &username, &isActive); err != nil {
			return res, err
		}

		if userID == nil {
			continue
		}

		members = append(members, domain.TeamMember{
			UserID:   *userID,
			Username: *username,
			IsActive: *isActive,
		})
	}

	if err := rows.Err(); err != nil {
		return res, err
	}

	if !foundTeam {
		return res, domain.NewDomainError(domain.ErrorCodeNotFound, "team not found")
	}

	return domain.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}
