package postgres

type Config struct {
	Host          string `env:"HOST" env-default:"localhost"`
	Port          string `env:"PORT" env-default:"5432"`
	DBName        string `env:"DB_NAME" env-required:"true"`
	User          string `env:"USER" env-required:"true"`
	Pass          string `env:"PASSWORD" env-required:"true"`
	MaxConns      int    `env:"MAX_CONNS" env-default:"10"`
	MigrationsDir string `env:"MIGRATIONS_DIR" env-default:"migrations"`
}
