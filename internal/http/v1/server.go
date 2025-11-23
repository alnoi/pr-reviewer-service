package v1

import (
	"github.com/alnoi/pr-reviewer-service/internal/usecase"
)

var _ (ServerInterface) = &ServerHandler{}

// ServerHandler — наша реализация ServerInterface из server_gen.go.
type ServerHandler struct {
	teamUC  usecase.TeamUseCase
	userUC  usecase.UserUseCase
	prUC    usecase.PRUseCase
	statsUC usecase.StatsUseCase
}

// NewServerHandler собирает HTTP-слой поверх юзкейсов.
func NewServerHandler(
	teamUC usecase.TeamUseCase,
	userUC usecase.UserUseCase,
	prUC usecase.PRUseCase,
	statsUC usecase.StatsUseCase,
) *ServerHandler {
	return &ServerHandler{
		teamUC:  teamUC,
		userUC:  userUC,
		prUC:    prUC,
		statsUC: statsUC,
	}
}
