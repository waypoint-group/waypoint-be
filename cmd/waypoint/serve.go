package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

type ServeCommand struct {
	Port        int    `name:"port" default:"8080" env:"WAYPOINT_HTTP_PORT" required:"" help:"HTTP listen port."`
	DatabaseURL string `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL."`
	Migrate     bool   `name:"migrate" env:"WAYPOINT_MIGRATE" help:"Run database migrations on startup."`
}

func (cmd *ServeCommand) Run() (runErr error) {
	cfg := waypoint.Config{
		Port:        cmd.Port,
		DatabaseURL: cmd.DatabaseURL,
		Migrate:     cmd.Migrate,
	}

	if cfg.Migrate {
		if err := Migrate(cfg.DatabaseURL); err != nil {
			return err
		}
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
