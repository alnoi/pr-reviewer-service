package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

// UpsertUsers — создаёт пользователей или обновляет username / is_active
func (r *UserRepository) UpsertUsers(ctx context.Context, teamName string, members []domain.TeamMember) error {
	const query = `
		INSERT INTO users (id, team_name, username, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET
			team_name = EXCLUDED.team_name,
			username = EXCLUDED.username,
			is_active = EXCLUDED.is_active
	`

	batch := &pgx.Batch{}
	for _, m := range members {
		batch.Queue(query, m.UserID, teamName, m.Username, m.IsActive)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range members {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

// GetUserByID возвращает пользователя по его id.
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	const q = `
		SELECT id, username, is_active, team_name
		FROM users
		WHERE id = $1
	`

	var (
		id       string
		username string
		isActive bool
		teamName string
	)

	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&id, &username, &isActive, &teamName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.NewDomainError(domain.ErrorCodeNotFound, "user not found")
		}
		return domain.User{}, err
	}

	return domain.User{
		UserID:   id,
		Username: username,
		IsActive: isActive,
		TeamName: teamName,
	}, nil
}

// SetUserIsActive переключает active-флаг и возвращает обновлённого пользователя.
func (r *UserRepository) SetUserIsActive(ctx context.Context, userID string, active bool) (domain.User, error) {
	const q = `
		UPDATE users
		SET is_active = $2
		WHERE id = $1
		RETURNING id, username, is_active, team_name
	`

	var (
		id       string
		username string
		isActive bool
		teamName string
	)

	err := r.pool.QueryRow(ctx, q, userID, active).Scan(&id, &username, &isActive, &teamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.NewDomainError(domain.ErrorCodeNotFound, "user not found")
		}
		return domain.User{}, err
	}

	return domain.User{
		UserID:   id,
		Username: username,
		IsActive: isActive,
		TeamName: teamName,
	}, nil
}

// GetTeamMembers возвращает участников команды. Если onlyActive = true, то только активных.
func (r *UserRepository) GetTeamMembers(ctx context.Context, teamName string, onlyActive bool) ([]domain.User, error) {
	query := `
		SELECT id, username, is_active
		FROM users
		WHERE team_name = $1
	`
	if onlyActive {
		query += ` AND is_active = true`
	}
	query += ` ORDER BY username`

	rows, err := r.pool.Query(ctx, query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)

	for rows.Next() {
		var (
			id       string
			username string
			isActive bool
		)

		if err := rows.Scan(&id, &username, &isActive); err != nil {
			return nil, err
		}

		users = append(users, domain.User{
			UserID:   id,
			Username: username,
			IsActive: isActive,
			TeamName: teamName,
		})
	}

	return users, rows.Err()
}
