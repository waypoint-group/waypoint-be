package authn

import (
	"context"
	"uuid"
)

type actorIDContextKey struct{}
type identityContextKey struct{}

// WithIdentityClaims stores claims after the caller has verified the token.
func WithIdentityClaims(ctx context.Context, claims *JWTClaims) context.Context {
	return context.WithValue(ctx, identityContextKey{}, claims)
}

// IdentityClaims returns the verified token claims, if present.
func IdentityClaims(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value(identityContextKey{}).(*JWTClaims)
	return claims, ok
}

// WithActorID stores the local actor resolved from the verified identity.
func WithActorID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, actorIDContextKey{}, id)
}

// ActorID returns the authenticated local actor, if present.
func ActorID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(actorIDContextKey{}).(uuid.UUID)
	return id, ok
}
