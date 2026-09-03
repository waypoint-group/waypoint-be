package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := os.Getenv("WAYPOINT_DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("WAYPOINT_DATABASE_URL environment variable is required")
	}

	port := 8080
	if value, ok := os.LookupEnv("WAYPOINT_HTTP_PORT"); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return fmt.Errorf("invalid WAYPOINT_HTTP_PORT %q: must be an integer from 1 to 65535", value)
		}
		port = parsed
	}

	cfg := waypoint.Config{
		Port:        port,
		DatabaseURL: databaseURL,
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
