package v1

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	applog "github.com/alnoi/pr-reviewer-service/internal/logger"
	"go.uber.org/zap"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

func (s *ServerHandler) PostTeamAdd(ctx echo.Context) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("PostTeamAdd called")

	var body PostTeamAddJSONRequestBody

	if err := ctx.Bind(&body); err != nil {
		log.Warn("invalid json in PostTeamAdd", zap.Error(err))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "invalid json")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	members := make([]domain.TeamMember, 0, len(body.Members))
	for _, m := range body.Members {
		members = append(members, domain.TeamMember{
			UserID:   m.UserId,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	team, err := s.teamUC.CreateTeam(ctx.Request().Context(), domain.Team{
		TeamName: body.TeamName,
		Members:  members,
	})
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

	return ctx.JSON(http.StatusCreated, map[string]any{
		"team": toAPITeam(team),
	})
}

func (s *ServerHandler) GetTeamGet(ctx echo.Context, params GetTeamGetParams) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("GetTeamGet called", zap.String("team_name", params.TeamName))

	if params.TeamName == "" {
		log.Warn("invalid data in GetTeamGet", zap.String("team_name", params.TeamName))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "team_name is required")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	team, err := s.teamUC.GetTeam(ctx.Request().Context(), params.TeamName)
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

	return ctx.JSON(http.StatusOK, toAPITeam(team))
}

func (s *ServerHandler) PostTeamDeactivateMembers(ctx echo.Context) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("PostTeamDeactivateMembers called")

	var body PostTeamDeactivateMembersJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		log.Warn("invalid json in PostTeamDeactivateMembers", zap.Error(err))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "invalid json")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	if body.TeamName == "" || len(body.UserIds) == 0 {
		log.Warn("invalid data in PostTeamDeactivateMembers",
			zap.String("team_name", body.TeamName),
			zap.Int("user_ids_count", len(body.UserIds)),
		)
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"),
			"team_name and user_ids are required")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	updatedTeam, err := s.teamUC.DeactivateTeamMembers(ctx.Request().Context(), body.TeamName, body.UserIds)
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

	return ctx.JSON(http.StatusOK, map[string]any{
		"team": toAPITeam(updatedTeam),
	})
}
