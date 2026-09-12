package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

// ServeCommand configures the Waypoint HTTP server command.
type ServeCommand struct {
	Port        int       `name:"port" default:"8080" env:"WAYPOINT_HTTP_PORT" required:"" help:"HTTP listen port."`
	DatabaseURL string    `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL."`
	Migrate     bool      `name:"migrate" env:"WAYPOINT_MIGRATE" help:"Run database migrations on startup."`
	JWT         JWTConfig `embed:"" prefix:"jwt." envprefix:"WAYPOINT_JWT_"`
}

// JWTConfig contains configuration parameters for JSON Web Token
// (JWT) validation.
type JWTConfig struct {
	Issuer   string `name:"issuer" env:"ISSUER" required:"" help:"Trusted JWT issuer URL."`
	Audience string `name:"audience" env:"AUDIENCE" required:"" help:"Required JWT audience."`
}

// Run starts the Waypoint HTTP server.
func (cmd *ServeCommand) Run() (runErr error) {
	OIDCProvider, err := middleware.DiscoverOIDCProvider(cmd.JWT.Issuer)
	if err != nil {
		return fmt.Errorf("discover OIDC provider: %w", err)
	}

	keyFunc, err := keyfunc.NewDefault([]string{OIDCProvider.JwksURI})
	if err != nil {
		return fmt.Errorf("create JWKS keyfunc: %w", err)
	}

	cfg := waypoint.Config{
		Port:        cmd.Port,
		DatabaseURL: cmd.DatabaseURL,
		Migrate:     cmd.Migrate,
		JWT: middleware.JWTConfig{
			Issuer:   cmd.JWT.Issuer,
			Audience: cmd.JWT.Audience,
			KeyFunc:  keyFunc.Keyfunc,
		},
	}

	app, err := waypoint.New(&cfg)
	if err != nil {
		return fmt.Errorf("create Waypoint app: %w", err)
	}
	defer app.Close()

	log.Printf("starting Waypoint server on port %d", cfg.Port)
	addr := ":" + strconv.Itoa(cfg.Port)
	server := http.Server{
		Addr:    addr,
		Handler: app.Handler,
	}
	return server.ListenAndServe()
}
