package logger

import (
	"context"
	"github.com/lufeed/feed-parser-api/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log *zap.Logger
)

func Initialize(cfg *config.AppConfig) error {
	environment := cfg.Service.Environment
	if environment == "production" {
		log = zap.Must(zap.NewProduction())
	} else {
		logConfig := zap.NewDevelopmentConfig()
		logLevel, err := zap.ParseAtomicLevel(cfg.Log.Level)
		if err != nil {
			return err
		}
		logConfig.Level = logLevel
		logConfig.DisableStacktrace = true
		logConfig.OutputPaths = []string{"stdout"}
		logConfig.ErrorOutputPaths = []string{"stdout"}
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		log, _ = logConfig.Build()
	}

	log = log.With(zap.String("service", cfg.Service.Name), zap.String("environment", environment))

	log.Info("Logger initialized")
	return nil
}

func GetLogger() *zap.Logger {
	return log
}

func GetSugaredLogger() *zap.SugaredLogger {
	return GetLogger().Sugar()
}

func GetLoggerFromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value("logger").(*zap.Logger); ok {
		return logger
	}
	return GetLogger()
}

func SetLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}
