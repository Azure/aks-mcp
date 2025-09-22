package logger

import (
	"github.com/sirupsen/logrus"
)

// Logger is the global logrus logger instance
var Logger = logrus.New()

func init() {
	// Set timestamp format to a shorter readable format
	Logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05",
		FullTimestamp:   true,
	})
}

// SetLevel sets the log level
func SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	Logger.SetLevel(logLevel)
	return nil
}

// GetLevel gets the current log level
func GetLevel() string {
	return Logger.GetLevel().String()
}

// Convenience functions using the global Logger
func Debug(args ...any) {
	Logger.Debug(args...)
}

func Debugf(format string, args ...any) {
	Logger.Debugf(format, args...)
}

func Info(args ...any) {
	Logger.Info(args...)
}

func Infof(format string, args ...any) {
	Logger.Infof(format, args...)
}

func Warn(args ...any) {
	Logger.Warn(args...)
}

func Warnf(format string, args ...any) {
	Logger.Warnf(format, args...)
}

func Error(args ...any) {
	Logger.Error(args...)
}

func Errorf(format string, args ...any) {
	Logger.Errorf(format, args...)
}
