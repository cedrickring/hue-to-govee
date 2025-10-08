package logger

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Default returns a new logger with the configured log level
func Default() zerolog.Logger {
	logger := zerolog.New(zerolog.NewConsoleWriter())

	level, err := zerolog.ParseLevel(viper.GetString("log_level"))
	if err != nil {
		logger.Error().Err(err).Msg("Invalid log level in config, defaulting to Info level")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	return logger
}
