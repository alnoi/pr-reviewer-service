package v1

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	applog "github.com/alnoi/pr-reviewer-service/internal/logger"
	"go.uber.org/zap"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

// POST /pullRequest/create
func (s *ServerHandler) PostPullRequestCreate(ctx echo.Context) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("PostPullRequestCreate called")

	var body PostPullRequestCreateJSONRequestBody

	if err := ctx.Bind(&body); err != nil {
		log.Warn("invalid json in PostPullRequestCreate", zap.Error(err))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "invalid json")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	if body.PullRequestId == "" || body.PullRequestName == "" || body.AuthorId == "" {
		log.Warn("invalid data in PostPullRequestCreate", zap.String("pull_request_id", body.PullRequestId), zap.String("pull_request_name", body.PullRequestName), zap.String("author_id", body.AuthorId))
		resp := newAPIError(
			ErrorResponseErrorCode("BAD_REQUEST"),
			"pull_request_id, pull_request_name and author_id are required",
		)
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	pr, err := s.prUC.CreatePR(
		ctx.Request().Context(),
		body.PullRequestId,
		body.PullRequestName,
		body.AuthorId,
	)
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
		"pr": toAPIPR(pr),
	})
}

// POST /pullRequest/merge
func (s *ServerHandler) PostPullRequestMerge(ctx echo.Context) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("PostPullRequestMerge called")

	var body PostPullRequestMergeJSONRequestBody

	if err := ctx.Bind(&body); err != nil {
		log.Warn("invalid json in PostPullRequestMerge", zap.Error(err))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "invalid json")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	if body.PullRequestId == "" {
		log.Warn("invalid data in PostPullRequestMerge", zap.String("pull_request_id", body.PullRequestId))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "pull_request_id is required")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	pr, err := s.prUC.MergePR(ctx.Request().Context(), body.PullRequestId)
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
		"pr": toAPIPR(pr),
	})
}

// POST /pullRequest/reassign
func (s *ServerHandler) PostPullRequestReassign(ctx echo.Context) error {
	log := applog.FromContext(ctx.Request().Context())
	log.Info("PostPullRequestReassign called")

	var body PostPullRequestReassignJSONRequestBody

	if err := ctx.Bind(&body); err != nil {
		log.Warn("invalid json in PostPullRequestReassign", zap.Error(err))
		resp := newAPIError(ErrorResponseErrorCode("BAD_REQUEST"), "invalid json")
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	if body.PullRequestId == "" || body.OldUserId == "" {
		log.Warn("invalid data in PostPullRequestReassign", zap.String("pull_request_id", body.PullRequestId), zap.String("old_user_id", body.OldUserId))
		resp := newAPIError(
			ErrorResponseErrorCode("BAD_REQUEST"),
			"pull_request_id and old_user_id are required",
		)
		return ctx.JSON(http.StatusBadRequest, resp)
	}

	pr, replacedBy, err := s.prUC.ReassignReviewer(
		ctx.Request().Context(),
		body.PullRequestId,
		body.OldUserId,
	)
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
		"pr":          toAPIPR(pr),
		"replaced_by": replacedBy,
	})
}
