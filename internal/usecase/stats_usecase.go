package usecase

import (
	"context"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (s *serviceImpl) GetStats(ctx context.Context) (domain.Stats, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.GetStats",
	)
	defer span.End()

	assignments, err := s.prRepo.GetAssignmentsCountByUser(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get assignment statistics")
		return domain.Stats{}, err
	}

	counts, err := s.prRepo.GetPRStatusCounts(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get PR status statistics")
		return domain.Stats{}, err
	}

	span.SetAttributes(
		attribute.Int("stats.assignments_users", len(assignments)),
		attribute.Int("stats.pr_total", counts.Total),
	)

	return domain.Stats{
		AssignmentsByUser: assignments,
		PRStatusCounts: domain.PRStatusCounts{
			Open:   counts.Open,
			Merged: counts.Merged,
			Total:  counts.Total,
		},
	}, nil
}
