package main

import (
	"context"
	"log"
	"net/http"
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"github.com/alnoi/pr-reviewer-service/config"
	dbpkg "github.com/alnoi/pr-reviewer-service/db"
	v1 "github.com/alnoi/pr-reviewer-service/internal/http/v1"
	"github.com/alnoi/pr-reviewer-service/internal/logger"
	"github.com/alnoi/pr-reviewer-service/internal/repository/postgres"
	"github.com/alnoi/pr-reviewer-service/internal/usecase"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	logg := logger.New()
	defer logg.Sync()
	zap.ReplaceGlobals(logg)

	// --- Observability setup ---

	if cfg.PyroscopeEnabled {
		go runPyroscope(logg, cfg.PyroscopeAddress)
	}

	var shutdownTracer func(context.Context) error
	if cfg.JaegerCollectorURL != "" {
		shutdownTracer = initTracer(logg, cfg.JaegerCollectorURL)
		defer func() {
			if err := shutdownTracer(ctx); err != nil {
				logg.Error("failed to shutdown tracer", zap.Error(err))
			}
		}()
	}

	go runMetricsServer(logg, cfg.MetricsPort)

	// --- App setup ---

	pool, err := pgxpool.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	defer pool.Close()

	dbpkg.SetupPostgres(pool, logg)

	teamRepo := postgres.NewTeamRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	prRepo := postgres.NewPRRepository(pool)

	transactor := dbpkg.NewTransactor(pool)

	useCase := usecase.NewService(teamRepo, userRepo, prRepo, transactor)

	handler := v1.NewServerHandler(useCase, useCase, useCase, useCase)

	r := v1.NewRouter(handler)
	r.Use(logger.Middleware(logg))

	if err := r.Start(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// --- Pyroscope ---

func runPyroscope(l *zap.Logger, addr string) {
	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: "pr-reviewer-service",
		ServerAddress:   addr,

		Logger: pyroscope.StandardLogger,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		l.Fatal("can not set up pyroscope", zap.Error(err))
	}
}

// --- Prometheus ---

func runMetricsServer(l *zap.Logger, port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	l.Info("starting metrics server", zap.String("port", port))

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		l.Fatal("can not start metrics server", zap.Error(err))
	}
}

// --- Tracing (Jaeger) ---

func initTracer(l *zap.Logger, url string) func(context.Context) error {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		l.Fatal("can not create jaeger collector", zap.Error(err))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("pr-reviewer-service"),
		)),
	)

	otel.SetTracerProvider(tp)

	l.Info("jaeger tracer initialized", zap.String("url", url))

	return tp.Shutdown
}
