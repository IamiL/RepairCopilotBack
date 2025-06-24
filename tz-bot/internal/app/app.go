package app

import (
	"log/slog"
	httpapp "repairCopilotBot/tz-bot/internal/app/http"
	tz_llm_client "repairCopilotBot/tz-bot/package/llm"
	tg_client "repairCopilotBot/tz-bot/package/tg"
	word_parser_client "repairCopilotBot/tz-bot/package/word-parser"
	"time"
)

type Config struct {
	TokenTTL time.Duration `yaml:"token_ttl" env-default:"300h"`
}

type App struct {
	HTTPServer *httpapp.App
}

func New(
	log *slog.Logger,
	appConfig *Config,
	httpConfig *httpapp.Config,
	LlmConfig *tz_llm_client.Config,
	WordParserConfig *word_parser_client.Config,
	TgConfig *tg_client.Config,
) *App {
	llmClient := tz_llm_client.New(LlmConfig.Url)

	wordParserClient := word_parser_client.New(WordParserConfig.Url)

	tgBot, err := tg_client.NewBot(TgConfig.Token)
	if err != nil {
		panic(err)
	}

	tgClient := tg_client.New(tgBot, TgConfig.ChatID)

	httpApp := httpapp.New(log, httpConfig, wordParserClient, llmClient, tgClient)

	return &App{
		HTTPServer: httpApp,
	}
}
