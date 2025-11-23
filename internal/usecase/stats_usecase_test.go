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

type statsDeps struct {
	prRepo *mock_usecase.MockPRRepository
}

func newStatsService(t *testing.T) (*serviceImpl, *statsDeps) {
	ctrl := gomock.NewController(t)

	deps := &statsDeps{
		prRepo: mock_usecase.NewMockPRRepository(ctrl),
	}

	s := &serviceImpl{
		prRepo: deps.prRepo,
	}

	return s, deps
}

func TestGetStats_Success(t *testing.T) {
	s, deps := newStatsService(t)
	ctx := context.Background()

	stats := []domain.UserAssignmentsStat{
		{UserID: "u1", ReviewAssignmentsCount: 3},
		{UserID: "u2", ReviewAssignmentsCount: 1},
	}

	counts := domain.PRStatusCounts{
		Open:   5,
		Merged: 10,
		Total:  15,
	}

	deps.prRepo.EXPECT().
		GetAssignmentsCountByUser(ctx).
		Return(stats, nil)

	deps.prRepo.EXPECT().
		GetPRStatusCounts(ctx).
		Return(counts, nil)

	res, err := s.GetStats(ctx)
	require.NoError(t, err)

	require.Equal(t, stats, res.AssignmentsByUser)
	require.Equal(t, counts.Open, res.PRStatusCounts.Open)
	require.Equal(t, counts.Merged, res.PRStatusCounts.Merged)
	require.Equal(t, counts.Total, res.PRStatusCounts.Total)
}

func TestGetStats_ErrorAssignments(t *testing.T) {
	s, deps := newStatsService(t)
	ctx := context.Background()

	wantErr := errors.New("assignments failed")

	deps.prRepo.EXPECT().
		GetAssignmentsCountByUser(ctx).
		Return(nil, wantErr)

	res, err := s.GetStats(ctx)
	require.Error(t, err)
	require.Equal(t, domain.Stats{}, res)
	require.ErrorIs(t, err, wantErr)
}

func TestGetStats_ErrorPRStatus(t *testing.T) {
	s, deps := newStatsService(t)
	ctx := context.Background()

	stats := []domain.UserAssignmentsStat{
		{UserID: "u1", ReviewAssignmentsCount: 3},
	}

	wantErr := errors.New("status failed")

	deps.prRepo.EXPECT().
		GetAssignmentsCountByUser(ctx).
		Return(stats, nil)

	deps.prRepo.EXPECT().
		GetPRStatusCounts(ctx).
		Return(domain.PRStatusCounts{}, wantErr)

	res, err := s.GetStats(ctx)
	require.Error(t, err)
	require.Equal(t, domain.Stats{}, res)
	require.ErrorIs(t, err, wantErr)
}
