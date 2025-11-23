package usecase

import (
	"context"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (s *serviceImpl) SetUserIsActive(ctx context.Context, userID string, isActive bool) (domain.User, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.SetUserIsActive",
		trace.WithAttributes(
			attribute.String("user.id", userID),
			attribute.Bool("user.is_active", isActive),
		),
	)
	defer span.End()

	user, err := s.userRepo.SetUserIsActive(ctx, userID, isActive)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to set user active status",
			zap.String("user_id", userID),
			zap.Bool("is_active", isActive),
		)
		return domain.User{}, err
	}

	return user, nil
}

func (s *serviceImpl) GetUserReviewPRs(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.GetUserReviewPRs",
		trace.WithAttributes(
			attribute.String("user.id", userID),
		),
	)
	defer span.End()

	if _, err := s.userRepo.GetUserByID(ctx, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get user",
			zap.String("user_id", userID),
		)
		return nil, err
	}

	prs, err := s.prRepo.GetPRsWhereReviewer(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get PRs where user is reviewer",
			zap.String("user_id", userID),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("user.review_prs_count", len(prs)))

	return prs, nil
}
