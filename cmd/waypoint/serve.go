package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/waypoint-group/waypoint-be/internal/middleware"
	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

// ServeCommand configures the Waypoint HTTP server command.
type ServeCommand struct {
	Port                     int       `name:"port" default:"8080" env:"WAYPOINT_HTTP_PORT" required:"" help:"HTTP listen port."`
	DatabaseURL              string    `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL."`
	Migrate                  bool      `name:"migrate" env:"WAYPOINT_MIGRATE" help:"Run database migrations on startup."`
	WorkspaceCleanupInterval string    `name:"workspace-cleanup-interval" default:"300s" env:"WAYPOINT_WORKSPACE_CLEANUP_INTERVAL" help:"Interval for cleaning up stale workspaces. Fractional values are supported. Must contain a unit suffix (e.g. 's' for seconds)"`
	JWT                      JWTConfig `embed:"" prefix:"jwt." envprefix:"WAYPOINT_JWT_"`
}

// JWTConfig contains configuration parameters for JSON Web Token
// (JWT) validation.
type JWTConfig struct {
	Issuer   string `name:"issuer" env:"ISSUER" required:"" help:"Trusted JWT issuer URL."`
	Audience string `name:"audience" env:"AUDIENCE" required:"" help:"Required JWT audience."`
}

// Run starts the Waypoint HTTP server.
func (cmd *ServeCommand) Run() (runErr error) {
	cfg, err := parseConfig(cmd)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	OIDCProvider, err := middleware.DiscoverOIDCProvider(cfg.JWT.Issuer)
	if err != nil {
		return fmt.Errorf("discover OIDC provider: %w", err)
	}

	keyFunc, err := keyfunc.NewDefault([]string{OIDCProvider.JwksURI})
	if err != nil {
		return fmt.Errorf("create JWKS keyfunc: %w", err)
	}
	cfg.JWT.KeyFunc = keyFunc.Keyfunc

	app, err := waypoint.New(cfg)
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

func parseConfig(cmd *ServeCommand) (*waypoint.Config, error) {
	if cmd.Port < 0 || cmd.Port > 65535 {
		return nil, fmt.Errorf("port must be between 0 and 65535")
	}

	if strings.TrimSpace(cmd.JWT.Issuer) == "" {
		return nil, fmt.Errorf("JWT issuer is required")
	}
	if strings.TrimSpace(cmd.JWT.Audience) == "" {
		return nil, fmt.Errorf("JWT audience is required")
	}

	WorkspaceCleanupInterval, err := time.ParseDuration(cmd.WorkspaceCleanupInterval)
	if err != nil {
		return nil, fmt.Errorf("parse workspace cleanup interval: %w", err)
	}
	if WorkspaceCleanupInterval <= 0 {
		return nil, fmt.Errorf("workspace cleanup interval must be positive")
	}

	return &waypoint.Config{
		Port:        cmd.Port,
		DatabaseURL: cmd.DatabaseURL,
		Migrate:     cmd.Migrate,
		JWT: middleware.JWTConfig{
			Issuer:   cmd.JWT.Issuer,
			Audience: cmd.JWT.Audience,
		},
		WorkspaceCleanupInterval: WorkspaceCleanupInterval,
	}, nil
}
