package log

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/env"
)

type prefork struct{}

func (p prefork) Run(e *zerolog.Event, level zerolog.Level, message string) {
	if fiber.IsChild() {
		e.Discard()
	}
}

func GetLogLevelFromEnv(envKey string, defaultLevel zerolog.Level) zerolog.Level {
	levelStr, err := env.Get(envKey, "")
	if err != nil {
		return defaultLevel
	}

	if levelStr == "" {
		return defaultLevel
	}

	levelStr = strings.ToUpper(levelStr)

	switch levelStr {
	case "TRACE":
		return zerolog.TraceLevel
	case "DEBUG":
		return zerolog.DebugLevel
	case "INFO":
		return zerolog.InfoLevel
	case "WARN", "WARNING":
		return zerolog.WarnLevel
	case "ERROR":
		return zerolog.ErrorLevel
	case "FATAL":
		return zerolog.FatalLevel
	case "PANIC":
		return zerolog.PanicLevel
	case "DISABLED", "NO", "OFF":
		return zerolog.Disabled
	default:
		panic("unknown log level: " + levelStr)
	}
}

func New() zerolog.Logger {
	return zerolog.New(
		func() io.Writer {
			logLevel := GetLogLevelFromEnv("LOG_LEVEL", zerolog.InfoLevel)
			zerolog.SetGlobalLevel(logLevel)
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

			writer := io.Writer(
				zerolog.ConsoleWriter{
					Out:        os.Stderr,
					TimeFormat: time.RFC3339Nano,
				},
			)

			return writer
		}(),
	).With().
		Timestamp().
		Caller().
		Logger().
		Hook(prefork{})
}
