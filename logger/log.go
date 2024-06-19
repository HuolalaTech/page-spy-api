package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func Log() *logrus.Logger {
	// logger.Level = logrus.DebugLevel
	logger.SetOutput(os.Stdout)
	return logger
}
