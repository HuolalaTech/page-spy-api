package logger

import "github.com/sirupsen/logrus"

var logger = logrus.New()

func Log() *logrus.Logger {
	// logger.Level = logrus.DebugLevel
	return logger
}
