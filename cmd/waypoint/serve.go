package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

type ServeCommand struct {
	DatabaseURL string `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL"`
	Port        int    `name:"port" env:"WAYPOINT_HTTP_PORT" required:"" help:"HTTP listen port."`
}

func (cmd *ServeCommand) Run() error {
	cfg := waypoint.Config{
		DatabaseURL: cmd.DatabaseURL,
		Port:        cmd.Port,
	}

	app, err := waypoint.New(&cfg)
	if err != nil {
		return err
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
