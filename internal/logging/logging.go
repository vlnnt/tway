package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New() (*zap.Logger, string, error) {
	logPath, err := Path()
	if err != nil {
		return nil, "", err
	}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, "", fmt.Errorf(
			"create log directory: %w",
			err,
		)
	}

	writer := zapcore.AddSync(
		&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     7,
			Compress:   true,
		},
	)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		writer,
		zap.InfoLevel,
	)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)

	return logger, logPath, nil
}

func Path() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf(
			"get user cache directory: %w",
			err,
		)
	}

	return filepath.Join(
		cacheDir,
		"tway",
		"tway.log",
	), nil
}
