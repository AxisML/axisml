package config

import "fmt"

// Common holds the process-wide database and logging configuration owned by
// the Compute Service binary. Keeping it module-local prevents the embeddable
// service library from depending on a separately released config helper.
type Common struct {
	Database Database `mapstructure:"database"`
	Log      Log      `mapstructure:"log"`
}

type Database struct {
	Host     string `mapstructure:"host" default:"localhost" doc:"PostgreSQL host"`
	Port     int    `mapstructure:"port" default:"5432" doc:"PostgreSQL port"`
	Name     string `mapstructure:"name" default:"axisml" doc:"Database name"`
	User     string `mapstructure:"user" default:"axisml" doc:"Database user"`
	Password string `mapstructure:"password" secret:"true" doc:"Database password"`
	SSLMode  string `mapstructure:"sslmode" default:"disable" doc:"libpq sslmode: disable | require | verify-full"`
}

type Log struct {
	Level  string `mapstructure:"level" default:"info" doc:"Log level: debug | info | warn | error"`
	Format string `mapstructure:"format" default:"json" doc:"Log format: json | console"`
}

func (d Database) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

func (d Database) URL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

func (c Common) PostgresDSN() string { return c.Database.DSN() }
func (c Common) PostgresURL() string { return c.Database.URL() }
func (l Log) Development() bool      { return l.Format == "console" }
