package v1

import (
	"errors"
	"github.com/alnoi/pr-reviewer-service/internal/domain"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetStats handles GET /stats
func (s *ServerHandler) GetStats(ctx echo.Context) error {
	reqCtx := ctx.Request().Context()

	stats, err := s.statsUC.GetStats(reqCtx)
	if err != nil {
		var derr *domain.DomainError
		if errors.As(err, &derr) {
			status := mapDomainErrorToStatus(derr.Code)
			resp := newAPIError(ErrorResponseErrorCode(derr.Code), derr.Error())
			return ctx.JSON(status, resp)
		}

		resp := newAPIError(ErrorResponseErrorCode("INTERNAL"), "internal server error")
		return ctx.JSON(http.StatusInternalServerError, resp)
	}

	apiStats := Stats{
		AssignmentsByUser: make([]UserAssignmentsStat, 0, len(stats.AssignmentsByUser)),
		PrStatusCounts: PRStatusCounts{
			Open:   int32(stats.PRStatusCounts.Open),
			Merged: int32(stats.PRStatusCounts.Merged),
			Total:  int32(stats.PRStatusCounts.Total),
		},
	}

	for _, sUser := range stats.AssignmentsByUser {
		apiStats.AssignmentsByUser = append(apiStats.AssignmentsByUser, UserAssignmentsStat{
			UserId:                 sUser.UserID,
			ReviewAssignmentsCount: int32(sUser.ReviewAssignmentsCount),
		})
	}

	return ctx.JSON(http.StatusOK, apiStats)
}
