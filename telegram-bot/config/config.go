package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env           string         `yaml:"env" env-default:"local"`
	Telegram      TelegramConfig `yaml:"telegram"`
	Postgres      PostgresConfig `yaml:"postgres"`
	UserService   ServiceConfig  `yaml:"user_service"`
	ChatService   ServiceConfig  `yaml:"chat_service"`
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token" env:"TELEGRAM_BOT_TOKEN"`
}

type PostgresConfig struct {
	Host           string `yaml:"host" env-default:"localhost"`
	Port           string `yaml:"port" env-default:"5432"`
	DatabaseName   string `yaml:"database_name" env-default:"telegram_bot"`
	Username       string `yaml:"username" env-default:"postgres"`
	Password       string `yaml:"password" env-default:"postgres"`
	MaxConnections int    `yaml:"max_connections" env-default:"10"`
}

type ServiceConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout" env-default:"10s"`
}

// MustLoad загружает конфигурацию из файла
func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

// MustLoadPath загружает конфигурацию из указанного файла
func MustLoadPath(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath получает путь к конфигурационному файлу из флага или переменной окружения
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}