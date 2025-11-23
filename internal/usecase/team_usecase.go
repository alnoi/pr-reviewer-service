package usecase

import (
	"context"
	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"github.com/alnoi/pr-reviewer-service/internal/logger"
	"github.com/alnoi/pr-reviewer-service/internal/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"math/rand"
)

type prUpdate struct {
	id        string
	reviewers []string
}

func (s *serviceImpl) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.CreateTeam",
		trace.WithAttributes(
			attribute.String("team.name", team.TeamName),
			attribute.Int("team.members_count", len(team.Members)),
		),
	)
	defer span.End()

	var res domain.Team

	err := s.transactor.WithTx(ctx, func(ctx context.Context) error {
		if err := s.teamRepo.CreateTeam(ctx, team.TeamName); err != nil {
			logger.LogDomainAware(ctx, err, "failed to create team inside transaction",
				zap.String("team_name", team.TeamName),
			)
			return err
		}

		if len(team.Members) > 0 {
			if err := s.userRepo.UpsertUsers(ctx, team.TeamName, team.Members); err != nil {
				logger.LogDomainAware(ctx, err, "failed to upsert team members inside transaction",
					zap.String("team_name", team.TeamName),
				)
				return err
			}
		}

		created, err := s.teamRepo.GetTeam(ctx, team.TeamName)
		if err != nil {
			logger.LogDomainAware(ctx, err, "failed to fetch created team inside transaction",
				zap.String("team_name", team.TeamName),
			)
			return err
		}

		res = created
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return domain.Team{}, err
	}

	metrics.TeamCreatedTotal.Inc()

	return res, nil
}

func (s *serviceImpl) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.GetTeam",
		trace.WithAttributes(
			attribute.String("team.name", teamName),
		),
	)
	defer span.End()

	team, err := s.teamRepo.GetTeam(ctx, teamName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get team",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}

	return team, nil
}

func (s *serviceImpl) DeactivateTeamMembers(ctx context.Context, teamName string, userIDs []string) (domain.Team, error) {
	ctx, span := tracer.Start(
		ctx,
		"Service.DeactivateTeamMembers",
		trace.WithAttributes(
			attribute.String("team.name", teamName),
			attribute.Int("deactivate.requested_count", len(userIDs)),
		),
	)
	defer span.End()

	team, err := s.teamRepo.GetTeam(ctx, teamName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get team before deactivation",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}

	if err := validateUsersInTeam(team, userIDs); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "attempt to deactivate user not in team",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}

	if len(userIDs) == 0 {
		return team, nil
	}

	toDeactivate, candidatePool, err := s.prepareDeactivationTargets(ctx, teamName, userIDs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to prepare deactivation targets",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}
	if len(toDeactivate) == 0 {
		return team, nil
	}

	prs, err := s.prRepo.GetOpenPRsByReviewers(ctx, toDeactivate)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to get open PRs for deactivated reviewers",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}

	if len(prs) == 0 {
		if err := s.applyDeactivationAndUpdates(ctx, toDeactivate, nil); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.LogDomainAware(ctx, err, "failed to apply deactivation for users without PRs",
				zap.String("team_name", teamName),
			)
			return domain.Team{}, err
		}
		span.SetAttributes(
			attribute.Int("deactivate.applied_count", len(toDeactivate)),
			attribute.Bool("deactivate.reassigned_prs", false),
		)
		return team, nil
	}

	if len(candidatePool) == 0 {
		derr := domain.NewDomainError(domain.ErrorCodeNoCandidate, "no active replacement candidate in team")
		span.RecordError(derr)
		span.SetStatus(codes.Error, derr.Error())
		logger.LogDomainAware(ctx, derr, "no replacement candidates for deactivated team members",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, derr
	}

	updates, err := s.preparePRUpdates(prs, candidatePool, toDeactivate)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to prepare PR updates for deactivation",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}

	if err := s.applyDeactivationAndUpdates(ctx, toDeactivate, updates); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.LogDomainAware(ctx, err, "failed to apply deactivation and PR updates",
			zap.String("team_name", teamName),
		)
		return domain.Team{}, err
	}

	span.SetAttributes(
		attribute.Int("deactivate.applied_count", len(toDeactivate)),
		attribute.Int("deactivate.updated_prs_count", len(updates)),
		attribute.Bool("deactivate.reassigned_prs", true),
	)

	updatedTeam, err := s.GetTeam(ctx, teamName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return domain.Team{}, err
	}

	metrics.TeamDeactivatedTotal.Inc()

	return updatedTeam, nil
}

// --------------------HELPERS-----------------------------

func validateUsersInTeam(team domain.Team, userIDs []string) error {
	memberSet := make(map[string]struct{}, len(team.Members))
	for _, m := range team.Members {
		memberSet[m.UserID] = struct{}{}
	}
	for _, id := range userIDs {
		if _, ok := memberSet[id]; !ok {
			return domain.NewDomainError(domain.ErrorCodeNotFound, "user not found in team")
		}
	}
	return nil
}

func buildUniqueIDSet(ids []string) map[string]struct{} {
	uniq := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		uniq[id] = struct{}{}
	}
	return uniq
}

func (s *serviceImpl) prepareDeactivationTargets(ctx context.Context, teamName string, userIDs []string) ([]string, []string, error) {
	uniq := buildUniqueIDSet(userIDs)

	members, err := s.userRepo.GetTeamMembers(ctx, teamName, true)
	if err != nil {
		return nil, nil, err
	}
	if len(members) == 0 {
		return nil, nil, nil
	}

	toDeactivateSet := make(map[string]struct{})
	toDeactivate := make([]string, 0, len(userIDs))
	for _, m := range members {
		if _, ok := uniq[m.UserID]; ok {
			toDeactivateSet[m.UserID] = struct{}{}
			toDeactivate = append(toDeactivate, m.UserID)
		}
	}

	if len(toDeactivate) == 0 {
		return nil, nil, nil
	}

	candidatePool := make([]string, 0, len(members))
	for _, m := range members {
		if _, disable := toDeactivateSet[m.UserID]; disable {
			continue
		}
		candidatePool = append(candidatePool, m.UserID)
	}

	return toDeactivate, candidatePool, nil
}

func buildBaseExclude(pr domain.PullRequest) map[string]struct{} {
	baseExclude := make(map[string]struct{}, len(pr.AssignedReviewers)+1)
	baseExclude[pr.AuthorID] = struct{}{}
	for _, rID := range pr.AssignedReviewers {
		baseExclude[rID] = struct{}{}
	}
	return baseExclude
}

func chooseReplacement(candidatePool []string, baseExclude map[string]struct{}) (string, error) {
	candidates := make([]string, 0, len(candidatePool))
	for _, cid := range candidatePool {
		if _, skip := baseExclude[cid]; skip {
			continue
		}
		candidates = append(candidates, cid)
	}

	if len(candidates) == 0 {
		return "", domain.NewDomainError(domain.ErrorCodeNoCandidate, "no active replacement candidate in team")
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	idx := rand.Intn(len(candidates))
	return candidates[idx], nil
}

func (s *serviceImpl) preparePRUpdates(prs []domain.PullRequest, candidatePool, toDeactivate []string) ([]prUpdate, error) {
	toDeactivateSet := make(map[string]struct{}, len(toDeactivate))
	for _, id := range toDeactivate {
		toDeactivateSet[id] = struct{}{}
	}

	updates := make([]prUpdate, 0, len(prs))

	for _, pr := range prs {
		if len(pr.AssignedReviewers) == 0 {
			continue
		}

		baseExclude := buildBaseExclude(pr)

		newReviewers := make([]string, len(pr.AssignedReviewers))
		copy(newReviewers, pr.AssignedReviewers)

		for i, rID := range pr.AssignedReviewers {
			if _, toDisable := toDeactivateSet[rID]; !toDisable {
				continue
			}

			chosen, err := chooseReplacement(candidatePool, baseExclude)
			if err != nil {
				return nil, err
			}

			newReviewers[i] = chosen
			baseExclude[chosen] = struct{}{}
		}

		updates = append(updates, prUpdate{
			id:        pr.PullRequestID,
			reviewers: newReviewers,
		})
	}

	return updates, nil
}

func (s *serviceImpl) applyDeactivationAndUpdates(ctx context.Context, toDeactivate []string, updates []prUpdate) error {
	return s.transactor.WithTx(ctx, func(txCtx context.Context) error {
		for _, id := range toDeactivate {
			if _, err := s.userRepo.SetUserIsActive(txCtx, id, false); err != nil {
				return err
			}
		}

		for _, u := range updates {
			if err := s.prRepo.SetPRReviewers(txCtx, u.id, u.reviewers); err != nil {
				return err
			}
		}

		return nil
	})
}
