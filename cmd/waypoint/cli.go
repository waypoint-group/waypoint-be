package main

type CLI struct {
	Serve   ServeCommand   `cmd:"" help:"Run the Waypoint HTTP server."`
	Migrate MigrateCommand `cmd:"" help:"Run database migrations."`
}
