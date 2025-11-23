//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	dbpkg "github.com/alnoi/pr-reviewer-service/db"
	v1 "github.com/alnoi/pr-reviewer-service/internal/http/v1"
	"github.com/alnoi/pr-reviewer-service/internal/logger"
	"github.com/alnoi/pr-reviewer-service/internal/repository/postgres"
	"github.com/alnoi/pr-reviewer-service/internal/usecase"

	"net/http/httptest"
)

var (
	dbPool      *pgxpool.Pool
	httpServer  *httptest.Server
	pgContainer testcontainers.Container
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		Env:          map[string]string{"POSTGRES_DB": "prreviewer", "POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}
	pgContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"test", "test", host, port.Port(), "prreviewer")

	dbPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pgx pool: %v\n", err)
		os.Exit(1)
	}

	logg := logger.New()
	defer logg.Sync()
	zap.ReplaceGlobals(logg)

	dbpkg.SetupPostgres(dbPool, logg)

	teamRepo := postgres.NewTeamRepository(dbPool)
	userRepo := postgres.NewUserRepository(dbPool)
	prRepo := postgres.NewPRRepository(dbPool)
	transactor := dbpkg.NewTransactor(dbPool)

	svc := usecase.NewService(teamRepo, userRepo, prRepo, transactor)

	handler := v1.NewServerHandler(svc, svc, svc, svc)
	e := v1.NewRouter(handler)
	e.Use(logger.Middleware(logg))

	httpServer = httptest.NewServer(e)

	code := m.Run()

	httpServer.Close()
	dbPool.Close()
	_ = pgContainer.Terminate(ctx)

	os.Exit(code)
}

func truncateAll(t *testing.T) {
	t.Helper()
	_, err := dbPool.Exec(context.Background(),
		`TRUNCATE TABLE pr_reviewers, pull_requests, users, teams RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
}

func TestTeamAddAndGet_E2E(t *testing.T) {
	truncateAll(t)

	teamReq := v1.Team{
		TeamName: "backend",
		Members: []v1.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
			{UserId: "u3", Username: "Charlie", IsActive: true},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(teamReq))

	resp, err := http.Post(httpServer.URL+"/team/add", "application/json", &buf)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created struct {
		Team v1.Team `json:"team"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	require.Equal(t, teamReq.TeamName, created.Team.TeamName)
	require.Len(t, created.Team.Members, len(teamReq.Members))

	getURL := httpServer.URL + "/team/get?team_name=" + url.QueryEscape(teamReq.TeamName)
	resp2, err := http.Get(getURL)
	require.NoError(t, err)
	defer resp2.Body.Close()

	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var got v1.Team
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&got))
	require.Equal(t, created.Team.TeamName, got.TeamName)
	require.Len(t, got.Members, len(teamReq.Members))
}

func TestCreatePRAndStats_E2E(t *testing.T) {
	truncateAll(t)

	teamReq := v1.Team{
		TeamName: "backend",
		Members: []v1.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
			{UserId: "u3", Username: "Charlie", IsActive: true},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(teamReq))

	resp, err := http.Post(httpServer.URL+"/team/add", "application/json", &buf)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	prReq := v1.PostPullRequestCreateJSONBody{
		AuthorId:        "u1",
		PullRequestId:   "pr-1",
		PullRequestName: "Test PR",
	}

	buf.Reset()
	require.NoError(t, json.NewEncoder(&buf).Encode(prReq))

	respPR, err := http.Post(httpServer.URL+"/pullRequest/create", "application/json", &buf)
	require.NoError(t, err)
	defer respPR.Body.Close()
	require.Equal(t, http.StatusCreated, respPR.StatusCode)

	var createdPR struct {
		Pr v1.PullRequest `json:"pr"`
	}
	require.NoError(t, json.NewDecoder(respPR.Body).Decode(&createdPR))
	prResp := createdPR.Pr
	require.Equal(t, prReq.PullRequestId, prResp.PullRequestId)
	require.Equal(t, prReq.PullRequestName, prResp.PullRequestName)
	require.Equal(t, prReq.AuthorId, prResp.AuthorId)
	require.Len(t, prResp.AssignedReviewers, 2)

	respStats, err := http.Get(httpServer.URL + "/stats")
	require.NoError(t, err)
	defer respStats.Body.Close()
	require.Equal(t, http.StatusOK, respStats.StatusCode)

	var stats v1.Stats
	require.NoError(t, json.NewDecoder(respStats.Body).Decode(&stats))

	require.Equal(t, int32(1), stats.PrStatusCounts.Total)
	require.Equal(t, int32(1), stats.PrStatusCounts.Open)
	require.Equal(t, int32(0), stats.PrStatusCounts.Merged)

	require.Len(t, stats.AssignmentsByUser, 3)

	var authorStat *v1.UserAssignmentsStat
	var totalAssignments int32
	for i := range stats.AssignmentsByUser {
		u := stats.AssignmentsByUser[i]
		if u.UserId == "u1" {
			authorStat = &u
		}
		totalAssignments += u.ReviewAssignmentsCount
	}

	require.NotNil(t, authorStat)
	require.Equal(t, int32(0), authorStat.ReviewAssignmentsCount)
	require.Equal(t, int32(2), totalAssignments)
}

func TestDeactivateMembers_NoCandidate_E2E(t *testing.T) {
	truncateAll(t)

	teamReq := v1.Team{
		TeamName: "backend",
		Members: []v1.TeamMember{
			{UserId: "u1", Username: "Author", IsActive: true},
			{UserId: "u2", Username: "OnlyReviewer", IsActive: true},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(teamReq))

	resp, err := http.Post(httpServer.URL+"/team/add", "application/json", &buf)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	prReq := v1.PostPullRequestCreateJSONBody{
		AuthorId:        "u1",
		PullRequestId:   "pr-2",
		PullRequestName: "PR for NO_CANDIDATE",
	}

	buf.Reset()
	require.NoError(t, json.NewEncoder(&buf).Encode(prReq))

	respPR, err := http.Post(httpServer.URL+"/pullRequest/create", "application/json", &buf)
	require.NoError(t, err)
	defer respPR.Body.Close()
	require.True(t, respPR.StatusCode >= 200 && respPR.StatusCode < 300)

	deactivateReq := v1.PostTeamDeactivateMembersJSONBody{
		TeamName: "backend",
		UserIds:  []string{"u2"},
	}

	buf.Reset()
	require.NoError(t, json.NewEncoder(&buf).Encode(deactivateReq))

	respDeact, err := http.Post(httpServer.URL+"/team/deactivateMembers", "application/json", &buf)
	require.NoError(t, err)
	defer respDeact.Body.Close()

	require.Equal(t, http.StatusConflict, respDeact.StatusCode)

	var errResp v1.ErrorResponse
	require.NoError(t, json.NewDecoder(respDeact.Body).Decode(&errResp))
	require.Equal(t, v1.NOCANDIDATE, errResp.Error.Code)
}
