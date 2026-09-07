package main

// CLI defines the commands exposed by the Waypoint command-line application.
type CLI struct {
	Serve   ServeCommand   `cmd:"" help:"Run the Waypoint HTTP server."`
	Migrate MigrateCommand `cmd:"" help:"Run database migrations."`
}
