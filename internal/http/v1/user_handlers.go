package v1

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
	applog "github.com/alnoi/pr-reviewer-service/internal/logger"
	"go.uber.org/zap"
)

// POST /users/setIsActive
func (s *ServerHandler) PostUsersSetIsActive(ctx echo.Context) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("PostUsersSetIsActive called")
	var body PostUsersSetIsActiveJSONRequestBody

	if err := ctx.Bind(&body); err != nil {
		log.Warn("invalid json in PostUsersSetIsActive", zap.Error(err))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "invalid json")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	if body.UserId == "" {
		log.Warn("invalid data in PostUsersSetIsActive", zap.String("user_id", body.UserId))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "user_id is required")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	user, err := s.userUC.SetUserIsActive(ctx.Request().Context(), body.UserId, body.IsActive)
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
		"user": toAPIUser(user),
	})
}

// GET /users/getReview
func (s *ServerHandler) GetUsersGetReview(ctx echo.Context, params GetUsersGetReviewParams) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("GetUsersGetReview called", zap.String("user_id", params.UserId))
	if params.UserId == "" {
		log.Warn("invalid data in GetUsersGetReview", zap.String("user_id", params.UserId))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "user_id is required")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	prs, err := s.userUC.GetUserReviewPRs(ctx.Request().Context(), params.UserId)
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

	items := make([]PullRequestShort, 0, len(prs))
	for _, pr := range prs {
		items = append(items, toAPIPRShort(pr))
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"user_id":       params.UserId,
		"pull_requests": items,
	})
}
