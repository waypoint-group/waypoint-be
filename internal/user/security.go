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
type identityContextKey struct{}
type authorizationCountContextKey struct{}

// ActorID returns the authenticated actor and whether it is present in context.
func ActorID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(actorIDContextKey{}).(uuid.UUID)
	return id, ok
}

func identityClaims(ctx context.Context) (*middleware.JWTClaims, bool) {
	claims, ok := ctx.Value(identityContextKey{}).(*middleware.JWTClaims)
	return claims, ok
}

// Security authenticates the operations protected by the OpenAPI contract.
type Security struct {
	users     *Service
	jwtConfig middleware.JWTConfig
}

func NewSecurity(users *Service, cfg middleware.JWTConfig) *Security {
	return &Security{users: users, jwtConfig: cfg}
}

// WithRequestMetadata preserves the header count because ogen passes only the
// selected bearer token to HandleBearerAuth. Authentication happens in the hook.
func (s *Security) WithRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), authorizationCountContextKey{}, len(r.Header.Values("Authorization")))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandleBearerAuth verifies the token once and resolves the local actor when needed.
func (s *Security) HandleBearerAuth(ctx context.Context, operation api.OperationName, token api.BearerAuth) (context.Context, error) {
	if count, ok := ctx.Value(authorizationCountContextKey{}).(int); !ok || count != 1 {
		return ctx, &api.HTTPError{Status: http.StatusUnauthorized, Message: "Unauthorized"}
	}
	_, claims, err := middleware.ValidateJWT(token.Token, s.jwtConfig)
	if err != nil {
		return ctx, &api.HTTPError{Status: http.StatusUnauthorized, Message: "Unauthorized"}
	}
	ctx = context.WithValue(ctx, identityContextKey{}, claims)
	switch operation {
	case api.CreateUserOperation, api.MeOperation:
		// Registration and profile lookup also support identities without a local user.
		return ctx, nil
	default:
		actor, err := s.users.ReadUserByIdentity(ctx, claims.Issuer, claims.Subject)
		if errors.Is(err, ErrNotFound) {
			return ctx, &api.HTTPError{Status: http.StatusForbidden, Message: ErrForbidden.Error()}
		}
		if err != nil {
			log.Printf("resolve actor identity: %v", err)
			return ctx, &api.HTTPError{Status: http.StatusInternalServerError, Message: "internal server error"}
		}
		return context.WithValue(ctx, actorIDContextKey{}, actor.ID), nil
	}
}
