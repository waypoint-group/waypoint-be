package waypoint

// Config contains the configuration used to construct a Waypoint application.
type Config struct {
	// Port is the HTTP port on which the server listens.
	Port int
	// DatabaseURL is the PostgreSQL connection URL.
	DatabaseURL string
	// Migrate controls whether database migrations are run before startup.
	Migrate bool
}
