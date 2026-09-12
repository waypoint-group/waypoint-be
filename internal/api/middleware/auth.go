package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig contains the trusted issuer, audience, and key resolver for
// access tokens. KeyFunc must return trusted RSA public keys; it must not
// trust keys supplied by the token.
type JWTConfig struct {
	// Issuer is the exact issuer URL accepted by the API.
	Issuer string
	// Audience is the audience required in access tokens for this API.
	Audience string
	// KeyFunc resolves the RSA public key used to verify a token's signature.
	KeyFunc jwt.Keyfunc
}

type OIDCProvider struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// DiscoverOIDCProvider fetches and parses the OpenID Connect discovery document.
func DiscoverOIDCProvider(issuer string) (*OIDCProvider, error) {
	c := http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := c.Get(issuer + "/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("request OIDC discovery: %w", err)
	}

	var provider OIDCProvider
	if err := json.NewDecoder(response.Body).Decode(&provider); err != nil {
		return nil, fmt.Errorf("decode OIDC discovery response: %w", err)
	}

	if provider.Issuer != issuer {
		return nil, fmt.Errorf(
			"issuer mismatch: got %q, want %q",
			provider.Issuer,
			issuer,
		)
	}

	return &provider, nil
}

func ValidateJWT(raw string, jwtConfig JWTConfig) (*jwt.Token, *jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		jwtConfig.KeyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(jwtConfig.Issuer),
		jwt.WithAudience(jwtConfig.Audience),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("parse token: %w", err)
	}

	if claims.Subject == "" {
		return nil, nil, fmt.Errorf("token missing subject claim")
	}
	if claims.Issuer != jwtConfig.Issuer {
		return nil, nil, fmt.Errorf("token issuer claim mismatch")
	}

	return token, claims, nil
}
