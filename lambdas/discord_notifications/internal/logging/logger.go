package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger define la interfaz para los métodos de registro de información y error.
type Logger interface {
	Info(msg string, fields ...zapcore.Field)  // Info registra un mensaje informativo.
	Error(msg string, fields ...zapcore.Field) // Error registra un mensaje de error.
}

// ZapLogger es una implementación de la interfaz Logger utilizando Zap Logger.
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger crea una nueva instancia de ZapLogger.
func NewZapLogger() (*ZapLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &ZapLogger{logger: logger}, nil
}

// Close cierra el logger.
func (l *ZapLogger) Close() error {
	err := l.logger.Sync()
	if err != nil && err.Error() != "sync /dev/stderr: invalid argument" {
		return err
	}
	return nil
}

// Info registra un mensaje informativo.
func (l *ZapLogger) Info(msg string, fields ...zapcore.Field) {
	l.logger.Info(msg, fields...)
}

// Error registra un mensaje de error.
func (l *ZapLogger) Error(msg string, fields ...zapcore.Field) {
	l.logger.Error(msg, fields...)
}
