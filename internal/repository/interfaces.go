package repository

import (
	"context"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

type (
	TeamRepository interface {
		CreateTeam(ctx context.Context, teamName string) error
		GetTeam(ctx context.Context, teamName string) (domain.Team, error)
	}

	UserRepository interface {
		UpsertUsers(ctx context.Context, teamName string, members []domain.TeamMember) error
		GetUserByID(ctx context.Context, userID string) (domain.User, error)
		SetUserIsActive(ctx context.Context, userID string, isActive bool) (domain.User, error)
		GetTeamMembers(ctx context.Context, teamName string, onlyActive bool) ([]domain.User, error)
	}

	PRRepository interface {
		CreatePR(ctx context.Context, pr domain.PullRequest) error
		PRExists(ctx context.Context, prID string) (bool, error)
		GetPR(ctx context.Context, prID string) (domain.PullRequest, error)
		UpdatePR(ctx context.Context, pr domain.PullRequest) error

		GetPRReviewers(ctx context.Context, prID string) ([]string, error)
		SetPRReviewers(ctx context.Context, prID string, reviewers []string) error

		GetPRsWhereReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
		GetOpenPRsByReviewers(ctx context.Context, userIDs []string) ([]domain.PullRequest, error)

		GetAssignmentsCountByUser(ctx context.Context) ([]domain.UserAssignmentsStat, error)
		GetPRStatusCounts(ctx context.Context) (domain.PRStatusCounts, error)
	}
)
