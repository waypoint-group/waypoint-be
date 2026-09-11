package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/waypoint-group/waypoint-be/internal/api/httpapi"
	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

// ServeCommand configures the Waypoint HTTP server command.
type ServeCommand struct {
	Port        int       `name:"port" default:"8080" env:"WAYPOINT_HTTP_PORT" required:"" help:"HTTP listen port."`
	DatabaseURL string    `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL."`
	Migrate     bool      `name:"migrate" env:"WAYPOINT_MIGRATE" help:"Run database migrations on startup."`
	JWT         JWTConfig `embed:"" prefix:"jwt." envprefix:"WAYPOINT_JWT_"`
}

type JWTConfig struct {
	Issuer   string `name:"issuer" env:"ISSUER" required:"" help:"Trusted JWT issuer URL."`
	Audience string `name:"audience" env:"AUDIENCE" required:"" help:"Required JWT audience."`
}

// Run starts the Waypoint HTTP server.
func (cmd *ServeCommand) Run() (runErr error) {
	jwksURI, err := fetchJwksURI(cmd.JWT.Issuer)
	if err != nil {
		return fmt.Errorf("fetch JWKS URI: %w", err)
	}

	keyFunc, err := keyfunc.NewDefault([]string{jwksURI})
	if err != nil {
		return fmt.Errorf("create JWKS keyfunc: %w", err)
	}

	cfg := waypoint.Config{
		Port:        cmd.Port,
		DatabaseURL: cmd.DatabaseURL,
		Migrate:     cmd.Migrate,
		JWT: httpapi.JWTConfig{
			Issuer:   cmd.JWT.Issuer,
			Audience: cmd.JWT.Audience,
			KeyFunc:  keyFunc.Keyfunc,
		},
	}

	app, err := waypoint.New(&cfg)
	if err != nil {
		return fmt.Errorf("create Waypoint app: %w", err)
	}
	defer app.Database.Close()

	log.Printf("starting Waypoint server on port %d", cfg.Port)
	addr := ":" + strconv.Itoa(cfg.Port)
	server := http.Server{
		Addr:    addr,
		Handler: app.Handler,
	}
	return server.ListenAndServe()
}

// fetchJwksURI fetches the JSON Web Key Set (JWKS) URI from the
// OIDC issuer's discovery endpoint.
func fetchJwksURI(issuer string) (string, error) {
	c := http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := c.Get(issuer + "/.well-known/openid-configuration")
	if err != nil {
		return "", fmt.Errorf("request OIDC discovery: %w", err)
	}

	var discoveryResponse struct {
		Issuer                string `json:"issuer"`
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		JwksURI               string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(response.Body).Decode(&discoveryResponse); err != nil {
		return "", fmt.Errorf("decode OIDC discovery response: %w", err)
	}

	if discoveryResponse.Issuer != issuer {
		return "", fmt.Errorf(
			"issuer mismatch: got %q, want %q",
			discoveryResponse.Issuer,
			issuer,
		)
	}

	return discoveryResponse.JwksURI, nil
}
