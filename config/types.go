package config

// Config of Postmoogle
type Config struct {
	// Homeserver url
	Homeserver string
	// Login is a MXID localpart (scheduler - OK, @scheduler:example.com - wrong)
	Login string
	// Password for login/password auth only
	Password string
	// Domain for SMTP
	Domain string
	// Port for SMTP
	Port string
	// RoomID of the admin room
	LogLevel string
	// NoEncryption disabled encryption support
	NoEncryption bool
	// Prefix for commands
	Prefix string
	// MaxSize of an email (including attachments)
	MaxSize int

	// DB config
	DB DB

	// Sentry config
	Sentry Sentry
}

// DB config
type DB struct {
	// DSN is a database connection string
	DSN string
	// Dialect of database, one of sqlite3, postgres
	Dialect string
}

// Sentry config
type Sentry struct {
	DSN        string
	SampleRate int
}
