package log

import (
	"fmt"

	"github.com/rs/zerolog"
)

var (
	logger *zerolog.Logger
)

func SetLogger(l *zerolog.Logger) {
	logger = l
}

// F prints debug log.
func F(f string, v ...any) {
	logger.Info().Msgf(f, v...)
}

// Print prints log.
func Print(v ...any) {
	logger.Warn().Msg(fmt.Sprint(v...))
}

// Printf prints log.
func Printf(f string, v ...any) {
	logger.Warn().Msgf(f, v...)
}

// Fatal log and exit.
func Fatal(v ...any) {
	logger.Fatal().Msg(fmt.Sprint(v...))
}

// Fatalf log and exit.
func Fatalf(f string, v ...any) {
	logger.Fatal().Msgf(f, v...)
}
