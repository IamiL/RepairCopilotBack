package postgres

type Config struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	DBName   string `yaml:"database_name"`
	User     string `yaml:"username"`
	Pass     string `yaml:"password"`
	MaxConns int    `yaml:"max_connections" default:"10"`
}
