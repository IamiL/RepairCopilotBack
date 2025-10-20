package repository

type RedisConfig struct {
	Address  string `env:"ADDR" env-default:"localhost:6379"`
	Password string `env:"PASS" env-default:""`
}