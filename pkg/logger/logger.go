package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Init(level, logFile string) error {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

	cores := []zapcore.Core{
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), lvl),
	}

	if logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err == nil {
			f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(f), lvl))
			}
		}
	}

	log = zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1))
	return nil
}

func L() *zap.Logger {
	if log == nil {
		log, _ = zap.NewDevelopment()
	}
	return log
}

func Info(msg string, fields ...zap.Field)  { L().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { L().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { L().Error(msg, fields...) }
func Debug(msg string, fields ...zap.Field) { L().Debug(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { L().Fatal(msg, fields...) }
