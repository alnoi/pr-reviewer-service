package v1

import (
	"net/http"

	"github.com/alnoi/pr-reviewer-service/internal/domain"
)

func mapDomainErrorToStatus(code domain.ErrorCode) int {
	switch code {
	case domain.ErrorCodeTeamExists:
		return http.StatusBadRequest
	case domain.ErrorCodePRExists:
		return http.StatusConflict
	case domain.ErrorCodePRMerged:
		return http.StatusConflict
	case domain.ErrorCodeNotAssigned:
		return http.StatusConflict
	case domain.ErrorCodeNoCandidate:
		return http.StatusConflict
	case domain.ErrorCodeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func newAPIError(code ErrorResponseErrorCode, msg string) ErrorResponse {
	return ErrorResponse{
		Error: struct {
			Code    ErrorResponseErrorCode `json:"code"`
			Message string                 `json:"message"`
		}{
			Code:    code,
			Message: msg,
		},
	}
}
