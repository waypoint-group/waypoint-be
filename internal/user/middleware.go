package user

import (
	"context"
	"errors"
	"log"
	"net/http"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/api"
	"github.com/waypoint-group/waypoint-be/internal/middleware"
)

type actorIDContextKey struct{}

// ActorID returns the authenticated actor and whether it is present in context.
func ActorID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(actorIDContextKey{}).(uuid.UUID)
	return id, ok
}

// RequireActor authenticates and resolves the local actor before calling a handler.
func (s *Service) RequireActor(cfg middleware.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, claims, err := middleware.ExtractAndValidateJWT(r, cfg)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			actor, err := s.ReadUserByIdentity(r.Context(), claims.Issuer, claims.Subject)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					http.Error(w, ErrForbidden.Error(), http.StatusForbidden)
				} else {
					log.Printf("resolve actor identity: %v", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
				return
			}
			ctx := context.WithValue(r.Context(), actorIDContextKey{}, actor.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type identityContextKey struct{}

func identityClaims(ctx context.Context) (*middleware.JWTClaims, bool) {
	claims, ok := ctx.Value(identityContextKey{}).(*middleware.JWTClaims)
	return claims, ok
}

// RequireIdentity verifies tokens for endpoints that do not require a local user.
func RequireIdentity(cfg middleware.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, claims, err := middleware.ExtractAndValidateJWT(r, cfg)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityContextKey{}, claims)))
		})
	}
}

// HandleBearerAuth bridges generated security to the outer authentication middleware.
func (s *Service) HandleBearerAuth(ctx context.Context, operation api.OperationName, _ api.BearerAuth) (context.Context, error) {
	switch operation {
	case api.CreateUserOperation, api.MeOperation:
		if _, ok := identityClaims(ctx); ok {
			return ctx, nil
		}
	default:
		if _, ok := ActorID(ctx); ok {
			return ctx, nil
		}
	}
	return ctx, errors.New("missing authenticated identity")
}
