package axismlconfig

import "fmt"

// Common holds the configuration sections shared by every AxisML service. A
// service embeds it with `mapstructure:",squash"` so the sections sit at the
// document root (database:, log:).
type Common struct {
	Database Database `mapstructure:"database"`
	Log      Log      `mapstructure:"log"`
}

// Database is the PostgreSQL connection. The password is supplied out of band
// (AXISML_DATABASE_PASSWORD or AXISML_DATABASE_PASSWORD_FILE), never the file.
type Database struct {
	Host     string `mapstructure:"host" default:"localhost" doc:"PostgreSQL host"`
	Port     int    `mapstructure:"port" default:"5432" doc:"PostgreSQL port"`
	Name     string `mapstructure:"name" default:"axisml" doc:"Database name"`
	User     string `mapstructure:"user" default:"axisml" doc:"Database user"`
	Password string `mapstructure:"password" secret:"true" doc:"Database password"`
	SSLMode  string `mapstructure:"sslmode" default:"disable" doc:"libpq sslmode: disable | require | verify-full"`
}

// Log configures structured logging.
type Log struct {
	Level  string `mapstructure:"level" default:"info" doc:"Log level: debug | info | warn | error"`
	Format string `mapstructure:"format" default:"json" doc:"Log format: json | console"`
}

// DSN returns a libpq-style DSN for GORM / pgx / golang-migrate.
func (d Database) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// URL returns a postgres:// URL form (golang-migrate prefers this).
func (d Database) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// PostgresDSN / PostgresURL are convenience pass-throughs so a Config that
// embeds Common keeps the call sites the services already use.
func (c Common) PostgresDSN() string { return c.Database.DSN() }
func (c Common) PostgresURL() string { return c.Database.URL() }

// Development reports whether logs should use the human-readable console format.
func (l Log) Development() bool { return l.Format == "console" }
