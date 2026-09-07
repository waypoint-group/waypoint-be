package main

type MigrateCommand struct {
	DatabaseURL string `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL"`
}

func (cmd *MigrateCommand) Run() error {
	// TODO: implement migration logic
	return nil
}
