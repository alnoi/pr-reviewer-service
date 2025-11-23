package usecase

import (
	"context"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/repository"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type (
	PRUseCase interface {
		CreatePR(ctx context.Context, prID, prName, authorID string) (domain.PullRequest, error)
		MergePR(ctx context.Context, prID string) (domain.PullRequest, error)
		ReassignReviewer(ctx context.Context, prID, oldUserID string) (pr domain.PullRequest, replacedBy string, err error)
	}

	StatsUseCase interface {
		GetStats(ctx context.Context) (domain.Stats, error)
	}

	TeamUseCase interface {
		CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error)
		GetTeam(ctx context.Context, teamName string) (domain.Team, error)

		// extra
		DeactivateTeamMembers(ctx context.Context, teamName string, userIDs []string) (domain.Team, error)
	}

	UserUseCase interface {
		SetUserIsActive(ctx context.Context, userID string, isActive bool) (domain.User, error)
		GetUserReviewPRs(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
	}

	Transactor interface {
		WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
)

var _ TeamUseCase = (*serviceImpl)(nil)
var _ UserUseCase = (*serviceImpl)(nil)
var _ PRUseCase = (*serviceImpl)(nil)
var _ StatsUseCase = (*serviceImpl)(nil)

var tracer = otel.Tracer("pr-reviewer-service")

type serviceImpl struct {
	logger     *zap.Logger
	teamRepo   repository.TeamRepository
	userRepo   repository.UserRepository
	prRepo     repository.PRRepository
	transactor Transactor
}

func NewService(
	teamRepo repository.TeamRepository,
	userRepo repository.UserRepository,
	prRepo repository.PRRepository,
	transactor Transactor,
) *serviceImpl {
	return &serviceImpl{
		teamRepo:   teamRepo,
		userRepo:   userRepo,
		prRepo:     prRepo,
		transactor: transactor,
	}
}
