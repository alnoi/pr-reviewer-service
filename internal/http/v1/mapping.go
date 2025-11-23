package v1

import (
	"time"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

// вспомогательная ф-ция для nullable time
func timePtr(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	tt := t.UTC()
	return &tt
}

func toAPITeam(t domain.Team) Team {
	members := make([]TeamMember, 0, len(t.Members))
	for _, m := range t.Members {
		members = append(members, TeamMember{
			UserId:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	return Team{
		TeamName: t.TeamName,
		Members:  members,
	}
}

func toAPIUser(u domain.User) User {
	return User{
		UserId:   u.UserID,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
}

func toAPIPR(pr domain.PullRequest) PullRequest {
	return PullRequest{
		PullRequestId:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorId:          pr.AuthorID,
		Status:            PullRequestStatus(pr.Status),
		AssignedReviewers: append([]string(nil), pr.AssignedReviewers...),
		CreatedAt:         timePtr(&pr.CreatedAt),
		MergedAt:          timePtr(pr.MergedAt),
	}
}

func toAPIPRShort(pr domain.PullRequestShort) PullRequestShort {
	return PullRequestShort{
		PullRequestId:   pr.PullRequestID,
		PullRequestName: pr.PullRequestName,
		AuthorId:        pr.AuthorID,
		Status:          PullRequestShortStatus(pr.Status),
	}
}
