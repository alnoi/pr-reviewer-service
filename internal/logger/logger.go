package logger

import (
	"context"
	"time"

	"errors"
	"github.com/alnoi/pr-reviewer-service/internal/domain"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ctxKey struct{}

func New() *zap.Logger {
	logger, _ := zap.NewDevelopment(zap.AddStacktrace(zap.ErrorLevel))
	return logger
}

func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func FromContext(ctx context.Context) *zap.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if lg, ok := v.(*zap.Logger); ok {
			return lg
		}
	}

	return zap.L()
}

func LogDomainAware(ctx context.Context, err error, msg string, fields ...zap.Field) {
	log := FromContext(ctx)

	var derr *domain.DomainError
	if errors.As(err, &derr) {
		log.Warn(msg,
			append(fields,
				zap.String("code", string(derr.Code)),
				zap.Error(err),
			)...,
		)
		return
	}

	log.Error(msg,
		append(fields, zap.Error(err))...,
	)
}

func Middleware(base *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			reqLogger := base.With(
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
			)

			ctx := WithContext(c.Request().Context(), reqLogger)
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			status := c.Response().Status
			latency := time.Since(start)

			if err != nil {
				reqLogger.Error("request finished",
					zap.Int("status", status),
					zap.Duration("latency", latency),
					zap.Error(err),
				)
			} else {
				reqLogger.Info("request finished",
					zap.Int("status", status),
					zap.Duration("latency", latency),
				)
			}

			return err
		}
	}
}
