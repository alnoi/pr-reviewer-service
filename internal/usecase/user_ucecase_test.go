package usecase

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestServiceImpl_SetUserIsActive_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	prRepo := mocks.NewMockPRRepository(ctrl)

	svc := &serviceImpl{
		userRepo: userRepo,
		prRepo:   prRepo,
	}

	ctx := context.Background()
	userID := "u1"
	isActive := true

	expectedUser := domain.User{
		UserID:   userID,
		Username: "Alice",
		TeamName: "backend",
		IsActive: isActive,
	}

	userRepo.
		EXPECT().
		SetUserIsActive(ctx, userID, isActive).
		Return(expectedUser, nil)

	user, err := svc.SetUserIsActive(ctx, userID, isActive)
	require.NoError(t, err)
	require.Equal(t, expectedUser, user)
}

func TestServiceImpl_SetUserIsActive_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	prRepo := mocks.NewMockPRRepository(ctrl)

	svc := &serviceImpl{
		userRepo: userRepo,
		prRepo:   prRepo,
	}

	ctx := context.Background()
	userID := "u1"
	isActive := false

	wantErr := errors.New("db error")

	userRepo.
		EXPECT().
		SetUserIsActive(ctx, userID, isActive).
		Return(domain.User{}, wantErr)

	user, err := svc.SetUserIsActive(ctx, userID, isActive)
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.User{}, user)
}

func TestServiceImpl_GetUserReviewPRs_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	prRepo := mocks.NewMockPRRepository(ctrl)

	svc := &serviceImpl{
		userRepo: userRepo,
		prRepo:   prRepo,
	}

	ctx := context.Background()
	userID := "u-missing"

	wantErr := errors.New("user not found")

	userRepo.
		EXPECT().
		GetUserByID(ctx, userID).
		Return(domain.User{}, wantErr)

	prRepo.
		EXPECT().
		GetPRsWhereReviewer(gomock.Any(), gomock.Any()).
		Times(0)

	prs, err := svc.GetUserReviewPRs(ctx, userID)
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, prs)
}

func TestServiceImpl_GetUserReviewPRs_RepoErrorOnPRs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	prRepo := mocks.NewMockPRRepository(ctrl)

	svc := &serviceImpl{
		userRepo: userRepo,
		prRepo:   prRepo,
	}

	ctx := context.Background()
	userID := "u1"

	userRepo.
		EXPECT().
		GetUserByID(ctx, userID).
		Return(domain.User{
			UserID:   userID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	wantErr := errors.New("pr repo error")

	prRepo.
		EXPECT().
		GetPRsWhereReviewer(ctx, userID).
		Return(nil, wantErr)

	prs, err := svc.GetUserReviewPRs(ctx, userID)
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, prs)
}

func TestServiceImpl_GetUserReviewPRs_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	prRepo := mocks.NewMockPRRepository(ctrl)

	svc := &serviceImpl{
		userRepo: userRepo,
		prRepo:   prRepo,
	}

	ctx := context.Background()
	userID := "u1"

	userRepo.
		EXPECT().
		GetUserByID(ctx, userID).
		Return(domain.User{
			UserID:   userID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	expectedPRs := []domain.PullRequestShort{
		{
			PullRequestID:   "pr-1",
			PullRequestName: "Add search",
			AuthorID:        "u2",
			Status:          domain.PRStatusOpen,
		},
		{
			PullRequestID:   "pr-2",
			PullRequestName: "Fix bug",
			AuthorID:        "u3",
			Status:          domain.PRStatusMerged,
		},
	}

	prRepo.
		EXPECT().
		GetPRsWhereReviewer(ctx, userID).
		Return(expectedPRs, nil)

	prs, err := svc.GetUserReviewPRs(ctx, userID)
	require.NoError(t, err)
	require.Len(t, prs, len(expectedPRs))

	for i := range expectedPRs {
		require.Equal(t, expectedPRs[i], prs[i])
	}
}
