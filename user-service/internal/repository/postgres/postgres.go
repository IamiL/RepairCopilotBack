package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string `env:"HOST" env-default:"localhost"`
	Port     string `env:"PORT" env-default:"5432"`
	DBName   string `env:"DB_NAME" env-required:"true"`
	User     string `env:"USER" env-required:"true"`
	Pass     string `env:"PASSWORD" env-required:"true"`
	MaxConns int    `env:"MAX_CONNS" env-default:"10"`
}

func (c *Config) ConnString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		c.User,
		c.Pass,
		c.Host,
		c.Port,
		c.DBName,
	)
}

func NewConnPool(config *Config) (*pgxpool.Pool, error) {
	pgxPollConfig, err := pgxpool.ParseConfig(config.ConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	pgxPollConfig.MaxConns = int32(config.MaxConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxPollConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}
