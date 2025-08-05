package repository

type RedisConfig struct {
	Address  string `yaml:"address" env-default:"localhost:6379"`
	Password string `yaml:"password" env-default:""`
}