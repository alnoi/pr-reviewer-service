package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	mock_usecase "github.com/alnoi/pr-reviewer-service/internal/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type teamDeps struct {
	teamRepo   *mock_usecase.MockTeamRepository
	userRepo   *mock_usecase.MockUserRepository
	prRepo     *mock_usecase.MockPRRepository
	transactor *mock_usecase.MockTransactor
}

func newTeamService(t *testing.T) (*serviceImpl, *teamDeps) {
	ctrl := gomock.NewController(t)

	deps := &teamDeps{
		teamRepo:   mock_usecase.NewMockTeamRepository(ctrl),
		userRepo:   mock_usecase.NewMockUserRepository(ctrl),
		prRepo:     mock_usecase.NewMockPRRepository(ctrl),
		transactor: mock_usecase.NewMockTransactor(ctrl),
	}

	s := &serviceImpl{
		teamRepo:   deps.teamRepo,
		userRepo:   deps.userRepo,
		prRepo:     deps.prRepo,
		transactor: deps.transactor,
	}

	return s, deps
}

func TestCreateTeam_SuccessWithMembers(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	input := domain.Team{
		TeamName: "team-1",
		Members: []domain.TeamMember{
			{UserID: "u1"},
			{UserID: "u2"},
		},
	}
	created := domain.Team{
		TeamName: input.TeamName,
		Members:  input.Members,
	}

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			err := f(txCtx)
			require.NoError(t, err)
			return nil
		})

	deps.teamRepo.EXPECT().
		CreateTeam(gomock.Any(), input.TeamName).
		Return(nil)

	deps.userRepo.EXPECT().
		UpsertUsers(gomock.Any(), input.TeamName, input.Members).
		Return(nil)

	deps.teamRepo.EXPECT().
		GetTeam(gomock.Any(), input.TeamName).
		Return(created, nil)

	res, err := s.CreateTeam(ctx, input)
	require.NoError(t, err)
	require.Equal(t, created, res)
}

func TestCreateTeam_SuccessWithoutMembers(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	input := domain.Team{
		TeamName: "team-2",
		Members:  nil,
	}
	created := domain.Team{
		TeamName: input.TeamName,
		Members:  nil,
	}

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			err := f(txCtx)
			require.NoError(t, err)
			return nil
		})

	deps.teamRepo.EXPECT().
		CreateTeam(gomock.Any(), input.TeamName).
		Return(nil)

	deps.teamRepo.EXPECT().
		GetTeam(gomock.Any(), input.TeamName).
		Return(created, nil)

	res, err := s.CreateTeam(ctx, input)
	require.NoError(t, err)
	require.Equal(t, created, res)
}

func TestCreateTeam_ErrorOnCreateTeam(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	input := domain.Team{
		TeamName: "team-err",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	wantErr := errors.New("create team failed")

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			return f(txCtx)
		})

	deps.teamRepo.EXPECT().
		CreateTeam(gomock.Any(), input.TeamName).
		Return(wantErr)

	res, err := s.CreateTeam(ctx, input)
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestCreateTeam_ErrorOnUpsertUsers(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	input := domain.Team{
		TeamName: "team-err-upsert",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	wantErr := errors.New("upsert users failed")

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			return f(txCtx)
		})

	deps.teamRepo.EXPECT().
		CreateTeam(gomock.Any(), input.TeamName).
		Return(nil)

	deps.userRepo.EXPECT().
		UpsertUsers(gomock.Any(), input.TeamName, input.Members).
		Return(wantErr)

	res, err := s.CreateTeam(ctx, input)
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestCreateTeam_ErrorOnGetTeam(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	input := domain.Team{
		TeamName: "team-err-get",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	wantErr := errors.New("get team failed")

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			return f(txCtx)
		})

	deps.teamRepo.EXPECT().
		CreateTeam(gomock.Any(), input.TeamName).
		Return(nil)

	deps.userRepo.EXPECT().
		UpsertUsers(gomock.Any(), input.TeamName, input.Members).
		Return(nil)

	deps.teamRepo.EXPECT().
		GetTeam(gomock.Any(), input.TeamName).
		Return(domain.Team{}, wantErr)

	res, err := s.CreateTeam(ctx, input)
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestGetTeam_Success(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	res, err := s.GetTeam(ctx, team.TeamName)
	require.NoError(t, err)
	require.Equal(t, team, res)
}

func TestGetTeam_Error(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	wantErr := errors.New("get team failed")

	deps.teamRepo.EXPECT().
		GetTeam(ctx, "team").
		Return(domain.Team{}, wantErr)

	res, err := s.GetTeam(ctx, "team")
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestDeactivateTeamMembers_EmptyUserIDs(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, nil)
	require.NoError(t, err)
	require.Equal(t, team, res)
}

func TestDeactivateTeamMembers_UserNotInTeam(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, []string{"u2"})
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorContains(t, err, "user not found in team")
}

func TestDeactivateTeamMembers_RepoGetTeamError(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	wantErr := errors.New("get team failed")

	deps.teamRepo.EXPECT().
		GetTeam(ctx, "team").
		Return(domain.Team{}, wantErr)

	res, err := s.DeactivateTeamMembers(ctx, "team", []string{"u1"})
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestDeactivateTeamMembers_NoActiveMembers(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	deps.userRepo.EXPECT().
		GetTeamMembers(ctx, team.TeamName, true).
		Return(nil, nil)

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, []string{"u1"})
	require.NoError(t, err)
	require.Equal(t, team, res)
}

func TestDeactivateTeamMembers_NoOpenPRs(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
			{UserID: "u2"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	deps.userRepo.EXPECT().
		GetTeamMembers(ctx, team.TeamName, true).
		Return([]domain.User{
			{UserID: "u1"},
			{UserID: "u2"},
		}, nil)

	deps.prRepo.EXPECT().
		GetOpenPRsByReviewers(ctx, []string{"u1"}).
		Return(nil, nil)

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			deps.userRepo.EXPECT().
				SetUserIsActive(txCtx, "u1", false).
				Return(domain.User{}, nil)
			return f(txCtx)
		})

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, []string{"u1"})
	require.NoError(t, err)
	require.Equal(t, team, res)
}

func TestDeactivateTeamMembers_NoCandidatePool(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	deps.userRepo.EXPECT().
		GetTeamMembers(ctx, team.TeamName, true).
		Return([]domain.User{
			{UserID: "u1"},
		}, nil)

	deps.prRepo.EXPECT().
		GetOpenPRsByReviewers(ctx, []string{"u1"}).
		Return([]domain.PullRequest{
			{
				PullRequestID:     "pr1",
				AuthorID:          "author",
				AssignedReviewers: []string{"u1"},
			},
		}, nil)

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, []string{"u1"})
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorContains(t, err, "no active replacement candidate in team")
}

func TestDeactivateTeamMembers_ErrorOnGetOpenPRs(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
			{UserID: "u2"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	deps.userRepo.EXPECT().
		GetTeamMembers(ctx, team.TeamName, true).
		Return([]domain.User{
			{UserID: "u1"},
			{UserID: "u2"},
		}, nil)

	wantErr := errors.New("get prs failed")

	deps.prRepo.EXPECT().
		GetOpenPRsByReviewers(ctx, []string{"u1"}).
		Return(nil, wantErr)

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, []string{"u1"})
	require.Error(t, err)
	require.Equal(t, domain.Team{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestDeactivateTeamMembers_SuccessWithPRReassign(t *testing.T) {
	s, deps := newTeamService(t)
	ctx := context.Background()

	team := domain.Team{
		TeamName: "team",
		Members: []domain.TeamMember{
			{UserID: "u1"},
			{UserID: "u2"},
			{UserID: "u3"},
		},
	}

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	deps.userRepo.EXPECT().
		GetTeamMembers(ctx, team.TeamName, true).
		Return([]domain.User{
			{UserID: "u1"},
			{UserID: "u2"},
			{UserID: "u3"},
		}, nil)

	prs := []domain.PullRequest{
		{
			PullRequestID:     "pr1",
			AuthorID:          "author",
			AssignedReviewers: []string{"u1", "u2"},
		},
	}

	deps.prRepo.EXPECT().
		GetOpenPRsByReviewers(ctx, []string{"u1"}).
		Return(prs, nil)

	deps.transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(txCtx context.Context, f func(context.Context) error) error {
			deps.userRepo.EXPECT().
				SetUserIsActive(txCtx, "u1", false).
				Return(domain.User{}, nil)

			deps.prRepo.EXPECT().
				SetPRReviewers(txCtx, "pr1", gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, reviewers []string) error {
					require.Len(t, reviewers, 2)
					require.NotContains(t, reviewers, "u1")
					return nil
				})

			return f(txCtx)
		})

	deps.teamRepo.EXPECT().
		GetTeam(ctx, team.TeamName).
		Return(team, nil)

	res, err := s.DeactivateTeamMembers(ctx, team.TeamName, []string{"u1"})
	require.NoError(t, err)
	require.Equal(t, team, res)
}
