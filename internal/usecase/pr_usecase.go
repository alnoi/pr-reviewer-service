package usecase

import (
	"context"
	"github.com/alnoi/pr-reviewer-service/internal/metrics"
	"math/rand"
	"time"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (s *serviceImpl) CreatePR(ctx context.Context, prID, prName, authorID string) (domain.PullRequest, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.CreatePR",
		trace.WithAttributes(
			attribute.String("pr.id", prID),
			attribute.String("pr.name", prName),
			attribute.String("pr.author_id", authorID),
		),
	)
	defer span.End()

	var res domain.PullRequest

	exists, err := s.prRepo.PRExists(ctx, prID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to check PR existence",
			zap.String("pr_id", prID),
		)
		return res, err
	}
	if exists {
		logger.FromContext(ctx).Warn("PR already exists", zap.String("pr_id", prID))
		derr := domain.NewDomainError(domain.ErrorCodePRExists, "PR id already exists")
		span.RecordError(derr)
		span.SetStatus(codes.Error, derr.Error())
		return res, derr
	}

	author, err := s.userRepo.GetUserByID(ctx, authorID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get PR author",
			zap.String("author_id", authorID),
		)
		return res, err
	}

	members, err := s.userRepo.GetTeamMembers(ctx, author.TeamName, true)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get team members for PR creation",
			zap.String("team", author.TeamName),
		)
		return res, err
	}

	exclude := map[string]struct{}{
		authorID: {},
	}
	candidateIDs := buildCandidateIDs(members, exclude)

	reviewers := shuffleAndTake(candidateIDs, 2)

	err = s.transactor.WithTx(ctx, func(txCtx context.Context) error {
		pr := domain.PullRequest{
			PullRequestID:     prID,
			PullRequestName:   prName,
			AuthorID:          authorID,
			Status:            domain.PRStatusOpen,
			AssignedReviewers: reviewers,
		}

		if err := s.prRepo.CreatePR(txCtx, pr); err != nil {
			logger.LogDomainAware(txCtx, err, "failed to create PR inside transaction",
				zap.String("pr_id", prID),
			)
			return err
		}

		if len(reviewers) > 0 {
			if err := s.prRepo.SetPRReviewers(txCtx, prID, reviewers); err != nil {
				logger.LogDomainAware(txCtx, err, "failed to set initial reviewers",
					zap.String("pr_id", prID),
				)
				return err
			}
		}

		created, err := s.prRepo.GetPR(txCtx, prID)
		if err != nil {
			logger.LogDomainAware(txCtx, err, "failed to fetch created PR",
				zap.String("pr_id", prID),
			)
			return err
		}

		res = created
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return res, err
	}

	span.SetAttributes(attribute.Int("pr.reviewers_count", len(res.AssignedReviewers)))

	metrics.PRCreatedTotal.Inc()

	return res, nil
}

func (s *serviceImpl) MergePR(ctx context.Context, prID string) (domain.PullRequest, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.MergePR",
		trace.WithAttributes(attribute.String("pr.id", prID)),
	)
	defer span.End()

	pr, err := s.prRepo.GetPR(ctx, prID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to fetch PR for merge",
			zap.String("pr_id", prID),
		)
		return domain.PullRequest{}, err
	}

	if pr.Status == domain.PRStatusMerged {
		logger.FromContext(ctx).Warn("PR already merged",
			zap.String("pr_id", prID),
		)
		return pr, nil
	}

	pr.Status = domain.PRStatusMerged
	now := time.Now()
	pr.MergedAt = &now

	if err := s.prRepo.UpdatePR(ctx, pr); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to update PR status to merged",
			zap.String("pr_id", prID),
		)
		return domain.PullRequest{}, err
	}

	span.SetAttributes(attribute.String("pr.status", string(pr.Status)))

	return pr, nil
}

func (s *serviceImpl) ReassignReviewer(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.ReassignReviewer",
		trace.WithAttributes(
			attribute.String("pr.id", prID),
			attribute.String("old_reviewer.id", oldUserID),
		),
	)
	defer span.End()

	pr, err := s.prRepo.GetPR(ctx, prID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to fetch PR for reassignment",
			zap.String("pr_id", prID),
		)
		return domain.PullRequest{}, "", err
	}

	if pr.Status == domain.PRStatusMerged {
		derr := domain.NewDomainError(domain.ErrorCodePRMerged, "cannot reassign on merged PR")
		span.RecordError(derr)
		span.SetStatus(codes.Error, derr.Error())
		logger.LogDomainAware(ctx, derr, "cannot reassign on merged PR",
			zap.String("pr_id", prID),
		)
		return domain.PullRequest{}, "", derr
	}

	oldUser, err := s.userRepo.GetUserByID(ctx, oldUserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to fetch old reviewer user",
			zap.String("user_id", oldUserID),
		)
		return domain.PullRequest{}, "", err
	}

	if !isReviewerAssigned(pr, oldUserID) {
		derr := domain.NewDomainError(domain.ErrorCodeNotAssigned, "reviewer is not assigned to this PR")
		span.RecordError(derr)
		span.SetStatus(codes.Error, derr.Error())
		logger.LogDomainAware(ctx, derr, "reviewer is not assigned to this PR",
			zap.String("pr_id", prID),
			zap.String("old_user_id", oldUserID),
		)
		return domain.PullRequest{}, "", derr
	}

	members, err := s.userRepo.GetTeamMembers(ctx, oldUser.TeamName, true)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to fetch team members for reassignment",
			zap.String("team", oldUser.TeamName),
		)
		return domain.PullRequest{}, "", err
	}

	exclude := make(map[string]struct{}, len(pr.AssignedReviewers)+1)
	exclude[pr.AuthorID] = struct{}{}
	for _, rID := range pr.AssignedReviewers {
		exclude[rID] = struct{}{}
	}

	candidateIDs := buildCandidateIDs(members, exclude)
	if len(candidateIDs) == 0 {
		derr := domain.NewDomainError(domain.ErrorCodeNoCandidate, "no active replacement candidate in team")
		span.RecordError(derr)
		span.SetStatus(codes.Error, derr.Error())
		logger.LogDomainAware(ctx, derr, "no active replacement candidate in team",
			zap.String("pr_id", prID),
		)
		return domain.PullRequest{}, "", derr
	}

	newReviewerID := chooseOneRandom(candidateIDs)

	newReviewers := replaceReviewer(pr.AssignedReviewers, oldUserID, newReviewerID)

	logger.FromContext(ctx).Debug("new reviewers", zap.Any("new_reviewers", newReviewers))

	if err := s.prRepo.SetPRReviewers(ctx, prID, newReviewers); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to update reviewers during reassignment",
			zap.String("pr_id", prID),
		)
		return domain.PullRequest{}, "", err
	}

	pr.AssignedReviewers = newReviewers

	span.SetAttributes(
		attribute.Int("reviewers.new_count", len(pr.AssignedReviewers)),
		attribute.String("reviewer.new_id", newReviewerID),
	)

	metrics.PRReassignedTotal.Inc()

	return pr, newReviewerID, nil
}

// --------------------HELPERS----------------------

func buildCandidateIDs(members []domain.User, exclude map[string]struct{}) []string {
	res := make([]string, 0, len(members))
	for _, m := range members {
		if _, skip := exclude[m.UserID]; skip {
			continue
		}
		res = append(res, m.UserID)
	}
	return res
}

func shuffleAndTake(ids []string, max int) []string {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 1 {
		rand.Shuffle(len(ids), func(i, j int) {
			ids[i], ids[j] = ids[j], ids[i]
		})
	}
	if len(ids) <= max {
		return ids
	}
	return ids[:max]
}

func chooseOneRandom(ids []string) string {
	if len(ids) == 1 {
		return ids[0]
	}
	idx := rand.Intn(len(ids))
	return ids[idx]
}

func isReviewerAssigned(pr domain.PullRequest, userID string) bool {
	for _, rID := range pr.AssignedReviewers {
		if rID == userID {
			return true
		}
	}
	return false
}

func replaceReviewer(reviewers []string, oldID, newID string) []string {
	res := make([]string, len(reviewers))
	copy(res, reviewers)
	for i, rID := range res {
		if rID == oldID {
			res[i] = newID
			break
		}
	}
	return res
}
