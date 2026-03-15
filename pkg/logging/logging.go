package logging

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger

func InitLogger(cfg LoggerConfig) {
	file := "app.log"
	if cfg.LogFile != "" {
		file = *&cfg.LogFile
	}
	logFile, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		panic(fmt.Sprintf("failed to open log file: %v", err))
	}
	multi := io.MultiWriter(os.Stdout, logFile)
	logger = zerolog.New(multi).With().Timestamp().Logger()
	switch cfg.LogFormat {
	case "json":
		return
	case "text":
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	default:
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	switch cfg.LogLevel {
	case "debug":
		logger = logger.Level(zerolog.DebugLevel)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		logger = logger.Level(zerolog.InfoLevel)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		logger = logger.Level(zerolog.WarnLevel)
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		logger = logger.Level(zerolog.ErrorLevel)
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		logger = logger.Level(zerolog.FatalLevel)
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		logger = logger.Level(zerolog.PanicLevel)
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		logger = logger.Level(zerolog.InfoLevel)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	logger.Hook(ReqIDHook{})
}

func L() zerolog.Logger {
	return logger
}

func FromCtx(ctx context.Context) zerolog.Logger {
	logger := ctx.Value("logger")
	if logger != nil {
		return logger.(zerolog.Logger)
	}
	return L()
}

type ReqIDHook struct{}

func (h ReqIDHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	userID, ok := e.GetCtx().Value("userID").(int)
	if ok {
		e.Int("user_id", userID)
	}
	requestID := e.GetCtx().Value("requestID").(string)
	if requestID != "" {
		e.Str("request_id", requestID)
	}
	operationID := e.GetCtx().Value("operationID").(string)
	if operationID != "" {
		e.Str("operation_id", operationID)
	}
}
func (h ReqIDHook) Levels() []zerolog.Level {
	return []zerolog.Level{
		zerolog.TraceLevel,
		zerolog.DebugLevel,
		zerolog.InfoLevel,
		zerolog.WarnLevel,
		zerolog.ErrorLevel,
		zerolog.FatalLevel,
		zerolog.PanicLevel,
	}
}
