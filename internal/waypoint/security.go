package waypoint

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/api"
	"github.com/waypoint-group/waypoint-be/internal/authn"
	"github.com/waypoint-group/waypoint-be/internal/user"
)

type authorizationCountContextKey struct{}

// apiSecurity applies authentication and local-account policy to API operations.
type apiSecurity struct {
	users     *user.Service
	jwtConfig authn.JWTConfig
}

var _ api.SecurityHandler = (*apiSecurity)(nil)

// WithRequestMetadata preserves the header count because ogen passes only the
// selected bearer token to HandleBearerAuth. Authentication happens in the hook.
func (s *apiSecurity) WithRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), authorizationCountContextKey{}, len(r.Header.Values("Authorization")))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandleBearerAuth verifies the token once and resolves the local actor when needed.
func (s *apiSecurity) HandleBearerAuth(ctx context.Context, operation api.OperationName, token api.BearerAuth) (context.Context, error) {
	if count, ok := ctx.Value(authorizationCountContextKey{}).(int); !ok || count != 1 {
		return ctx, &api.HTTPError{Status: http.StatusUnauthorized, Message: "Unauthorized"}
	}
	_, claims, err := authn.ValidateJWT(token.Token, s.jwtConfig)
	if err != nil {
		return ctx, &api.HTTPError{Status: http.StatusUnauthorized, Message: "Unauthorized"}
	}
	ctx = authn.WithIdentityClaims(ctx, claims)
	switch operation {
	case api.CreateUserOperation, api.MeOperation:
		// Registration and profile lookup also support identities without a local user.
		return ctx, nil
	default:
		actor, err := s.users.ReadUserByIdentity(ctx, claims.Issuer, claims.Subject)
		if errors.Is(err, user.ErrNotFound) {
			return ctx, &api.HTTPError{Status: http.StatusForbidden, Message: user.ErrForbidden.Error()}
		}
		if err != nil {
			log.Printf("resolve actor identity: %v", err)
			return ctx, &api.HTTPError{Status: http.StatusInternalServerError, Message: "internal server error"}
		}
		return authn.WithActorID(ctx, actor.ID), nil
	}
}
