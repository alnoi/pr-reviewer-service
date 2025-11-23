package usecase

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/mocks"
)

// ----------HELPERS FOR TESTS----------

func newPRServiceWithRepos(ctrl *gomock.Controller) (*serviceImpl, *mocks.MockPRRepository, *mocks.MockUserRepository, *mocks.MockTransactor) {
	prRepo := mocks.NewMockPRRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	tx := mocks.NewMockTransactor(ctrl)

	svc := &serviceImpl{
		prRepo:     prRepo,
		userRepo:   userRepo,
		transactor: tx,
	}

	return svc, prRepo, userRepo, tx
}

func isEqualPR(this, other domain.PullRequest) bool {
	return this.PullRequestID == other.PullRequestID &&
		this.PullRequestName == other.PullRequestName &&
		this.AuthorID == other.AuthorID &&
		this.Status == other.Status &&
		slices.Equal(this.AssignedReviewers, other.AssignedReviewers) &&
		this.CreatedAt == other.CreatedAt &&
		this.MergedAt == other.MergedAt
}

// ----------CREATE PR TESTS----------

func TestCreatePR_PrExistsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	wantErr := errors.New("db error")

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, wantErr)

	res, err := svc.CreatePR(ctx, prID, "name", "u1")

	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_PrAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(true, nil)

	res, err := svc.CreatePR(ctx, prID, "name", "u1")

	require.Error(t, err)

	var derr *domain.DomainError
	require.ErrorAs(t, err, &derr)
	require.Equal(t, domain.ErrorCodePRExists, derr.Code)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_GetAuthorError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	wantErr := errors.New("author not found")

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{}, wantErr)

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_GetTeamMembersError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{
			UserID:   authorID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	wantErr := errors.New("team error")

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return(nil, wantErr)

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_Tx_CreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, tx := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{
			UserID:   authorID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{}, nil)

	wantErr := errors.New("create failed")

	prRepo.
		EXPECT().
		CreatePR(gomock.Any(), gomock.Any()).
		Return(wantErr)

	tx.
		EXPECT().
		WithTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_Tx_SetReviewersError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, tx := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{
			UserID:   authorID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{
			{
				UserID:   "u2",
				Username: "Bob",
				TeamName: "backend",
				IsActive: true,
			},
		}, nil)

	prRepo.
		EXPECT().
		CreatePR(gomock.Any(), gomock.Any()).
		Return(nil)

	wantErr := errors.New("set reviewers failed")

	prRepo.
		EXPECT().
		SetPRReviewers(gomock.Any(), prID, []string{"u2"}).
		Return(wantErr)

	tx.
		EXPECT().
		WithTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_Tx_GetPR_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, tx := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{
			UserID:   authorID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{}, nil)

	prRepo.
		EXPECT().
		CreatePR(gomock.Any(), gomock.Any()).
		Return(nil)

	wantErr := errors.New("get pr failed")

	prRepo.
		EXPECT().
		GetPR(gomock.Any(), prID).
		Return(domain.PullRequest{}, wantErr)

	tx.
		EXPECT().
		WithTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.True(t, isEqualPR(res, domain.PullRequest{}))
}

func TestCreatePR_Success_NoReviewers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, tx := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{
			UserID:   authorID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{}, nil)

	prRepo.
		EXPECT().
		CreatePR(gomock.Any(), gomock.Any()).
		Return(nil)

	expected := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          authorID,
		Status:            domain.PRStatusOpen,
		AssignedReviewers: nil,
	}

	prRepo.
		EXPECT().
		GetPR(gomock.Any(), prID).
		Return(expected, nil)

	tx.
		EXPECT().
		WithTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.NoError(t, err)
	require.True(t, isEqualPR(res, expected))
}

func TestCreatePR_Success_WithReviewers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, tx := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	prRepo.
		EXPECT().
		PRExists(ctx, prID).
		Return(false, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, authorID).
		Return(domain.User{
			UserID:   authorID,
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{
			{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		}, nil)

	prRepo.
		EXPECT().
		CreatePR(gomock.Any(), gomock.Any()).
		Return(nil)

	prRepo.
		EXPECT().
		SetPRReviewers(gomock.Any(), prID, []string{"u2"}).
		Return(nil)

	expected := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          authorID,
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{"u2"},
	}

	prRepo.
		EXPECT().
		GetPR(gomock.Any(), prID).
		Return(expected, nil)

	tx.
		EXPECT().
		WithTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	res, err := svc.CreatePR(ctx, prID, "name", authorID)

	require.NoError(t, err)
	require.True(t, isEqualPR(res, expected))
}

// ----------MERGE PR TESTS----------

func TestMergePR_GetPRError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	wantErr := errors.New("db error")

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(domain.PullRequest{}, wantErr)

	_, err := svc.MergePR(ctx, prID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestMergePR_AlreadyMerged_Idempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	existing := domain.PullRequest{
		PullRequestID:   prID,
		PullRequestName: "name",
		AuthorID:        "u1",
		Status:          domain.PRStatusMerged,
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(existing, nil)

	prRepo.
		EXPECT().
		UpdatePR(gomock.Any(), gomock.Any()).
		Times(0)

	res, err := svc.MergePR(ctx, prID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isEqualPR(res, existing) {
		t.Fatalf("expected same PR, got %+v", res)
	}
}

func TestMergePR_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	existing := domain.PullRequest{
		PullRequestID:   prID,
		PullRequestName: "name",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(existing, nil)

	prRepo.
		EXPECT().
		UpdatePR(ctx, gomock.Any()).
		Return(nil)

	res, err := svc.MergePR(ctx, prID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Status != domain.PRStatusMerged {
		t.Fatalf("expected status MERGED, got %s", res.Status)
	}
	if res.MergedAt == nil {
		t.Fatalf("expected MergedAt to be set")
	}
	if time.Since(*res.MergedAt) > time.Second {
		t.Fatalf("expected recent MergedAt, got %v", res.MergedAt)
	}
}

// ----------REASSIGN REVIEWER TESTS----------

func TestReassignReviewer_GetPRError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	wantErr := errors.New("db error")

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(domain.PullRequest{}, wantErr)

	_, _, err := svc.ReassignReviewer(ctx, prID, "u2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestReassignReviewer_PRMerged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, _, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"

	pr := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          "u1",
		Status:            domain.PRStatusMerged,
		AssignedReviewers: []string{"u2"},
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(pr, nil)

	_, _, err := svc.ReassignReviewer(ctx, prID, "u2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var derr *domain.DomainError
	if !errors.As(err, &derr) || derr.Code != domain.ErrorCodePRMerged {
		t.Fatalf("expected PR_MERGED error, got %v", err)
	}
}

func TestReassignReviewer_GetOldUserError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	oldID := "u2"

	pr := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          "u1",
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{oldID},
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(pr, nil)

	wantErr := errors.New("user not found")

	userRepo.
		EXPECT().
		GetUserByID(ctx, oldID).
		Return(domain.User{}, wantErr)

	_, _, err := svc.ReassignReviewer(ctx, prID, oldID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestReassignReviewer_NotAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	oldID := "u999"

	pr := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          "u1",
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{"u2", "u3"},
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(pr, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, oldID).
		Return(domain.User{
			UserID:   oldID,
			Username: "Ghost",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	_, _, err := svc.ReassignReviewer(ctx, prID, oldID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var derr *domain.DomainError
	if !errors.As(err, &derr) || derr.Code != domain.ErrorCodeNotAssigned {
		t.Fatalf("expected NOT_ASSIGNED error, got %v", err)
	}
}

func TestReassignReviewer_GetTeamMembersError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	oldID := "u2"

	pr := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          "u1",
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{oldID},
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(pr, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, oldID).
		Return(domain.User{
			UserID:   oldID,
			Username: "Bob",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	wantErr := errors.New("team error")

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return(nil, wantErr)

	_, _, err := svc.ReassignReviewer(ctx, prID, oldID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestReassignReviewer_NoCandidates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	oldID := "u2"

	pr := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          "u1",
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{oldID},
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(pr, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, oldID).
		Return(domain.User{
			UserID:   oldID,
			Username: "Bob",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{
			{UserID: "u1", Username: "Author", TeamName: "backend", IsActive: true},
			{UserID: oldID, Username: "Bob", TeamName: "backend", IsActive: true},
		}, nil)

	_, _, err := svc.ReassignReviewer(ctx, prID, oldID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var derr *domain.DomainError
	if !errors.As(err, &derr) || derr.Code != domain.ErrorCodeNoCandidate {
		t.Fatalf("expected NO_CANDIDATE error, got %v", err)
	}
}

func TestReassignReviewer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, prRepo, userRepo, _ := newPRServiceWithRepos(ctrl)

	ctx := context.Background()
	prID := "pr-1"
	oldID := "u2"

	pr := domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   "name",
		AuthorID:          "u1",
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{oldID, "u3"},
	}

	prRepo.
		EXPECT().
		GetPR(ctx, prID).
		Return(pr, nil)

	userRepo.
		EXPECT().
		GetUserByID(ctx, oldID).
		Return(domain.User{
			UserID:   oldID,
			Username: "Bob",
			TeamName: "backend",
			IsActive: true,
		}, nil)

	userRepo.
		EXPECT().
		GetTeamMembers(ctx, "backend", true).
		Return([]domain.User{
			{UserID: "u1", Username: "Author", TeamName: "backend", IsActive: true},
			{UserID: oldID, Username: "Bob", TeamName: "backend", IsActive: true},
			{UserID: "u3", Username: "Carol", TeamName: "backend", IsActive: true},
			{UserID: "u4", Username: "Dave", TeamName: "backend", IsActive: true},
		}, nil)

	prRepo.
		EXPECT().
		SetPRReviewers(ctx, prID, []string{"u4", "u3"}).
		Return(nil)

	res, replacedBy, err := svc.ReassignReviewer(ctx, prID, oldID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if replacedBy != "u4" {
		t.Fatalf("expected replacedBy u4, got %s", replacedBy)
	}
	if len(res.AssignedReviewers) != 2 {
		t.Fatalf("expected 2 reviewers, got %d", len(res.AssignedReviewers))
	}
	if res.AssignedReviewers[0] != "u4" || res.AssignedReviewers[1] != "u3" {
		t.Fatalf("unexpected reviewers %+v", res.AssignedReviewers)
	}
}

// ----------HELPER FUNCTION TESTS----------

func TestBuildCandidateIDs(t *testing.T) {
	members := []domain.User{
		{UserID: "u1"},
		{UserID: "u2"},
		{UserID: "u3"},
	}
	exclude := map[string]struct{}{
		"u2": {},
	}
	got := buildCandidateIDs(members, exclude)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0] == "u2" || got[1] == "u2" {
		t.Fatalf("excluded user present in candidates: %+v", got)
	}
}

func TestShuffleAndTake_Empty(t *testing.T) {
	res := shuffleAndTake(nil, 2)
	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestShuffleAndTake_LessOrEqualMax(t *testing.T) {
	ids := []string{"u1"}
	res := shuffleAndTake(ids, 2)
	if len(res) != 1 || res[0] != "u1" {
		t.Fatalf("expected [u1], got %+v", res)
	}
}

func TestShuffleAndTake_MoreThanMax(t *testing.T) {
	ids := []string{"u1", "u2", "u3"}
	res := shuffleAndTake(ids, 2)
	if len(res) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(res))
	}
}

func TestIsReviewerAssigned(t *testing.T) {
	pr := domain.PullRequest{
		AssignedReviewers: []string{"u1", "u2"},
	}
	if !isReviewerAssigned(pr, "u1") {
		t.Fatalf("expected u1 to be assigned")
	}
	if isReviewerAssigned(pr, "u3") {
		t.Fatalf("expected u3 to not be assigned")
	}
}

func TestReplaceReviewer(t *testing.T) {
	revs := []string{"u1", "u2"}
	res := replaceReviewer(revs, "u2", "u3")

	if len(res) != 2 {
		t.Fatalf("expected len 2, got %d", len(res))
	}
	if res[0] != "u1" || res[1] != "u3" {
		t.Fatalf("unexpected result: %+v", res)
	}

	if revs[1] != "u2" {
		t.Fatalf("original slice modified: %+v", revs)
	}
}
