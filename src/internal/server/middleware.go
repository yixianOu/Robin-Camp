package server

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// AuthMiddleware validates Bearer token for write operations
func AuthMiddleware(token string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Get transport info
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.Unauthorized("UNAUTHORIZED", "missing transport info")
			}

			// Only apply auth to CreateMovie (NOT SubmitRating, it uses X-Rater-Id only)
			if tr.Operation() == "/api.movie.v1.MovieService/CreateMovie" {

				// Extract Authorization header
				authHeader := tr.RequestHeader().Get("Authorization")
				if authHeader == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "missing Authorization header")
				}

				// Check Bearer token format
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || parts[0] != "Bearer" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "invalid Authorization header format")
				}

				// Validate token
				if parts[1] != token {
					return nil, errors.Unauthorized("UNAUTHORIZED", "invalid token")
				}
			}

			return handler(ctx, req)
		}
	}
}

// RaterIdMiddleware extracts X-Rater-Id header for rating operations
func RaterIdMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Get transport info
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// Only apply to SubmitRating operation
			if tr.Operation() == "/api.movie.v1.MovieService/SubmitRating" {
				// Extract X-Rater-Id header
				raterID := tr.RequestHeader().Get("X-Rater-Id")
				if raterID == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "missing X-Rater-Id header")
				}

				// Inject rater ID into context
				ctx = context.WithValue(ctx, "rater_id", raterID)
			}

			return handler(ctx, req)
		}
	}
}
