package log

import (
	"io"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type prefork struct{}

func (p prefork) Run(e *zerolog.Event, level zerolog.Level, message string) {
	if fiber.IsChild() {
		e.Discard()
	}
}

func New() zerolog.Logger {
	return zerolog.New(
		func() io.Writer {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
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
