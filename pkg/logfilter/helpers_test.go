package logfilter_test

import (
	"time"

	"github.com/google/uuid"
	"github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"
)

func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = ginkgo.GinkgoWriter
	logger.Level = logrus.DebugLevel

	formatter := &logrus.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	logger.Formatter = formatter

	return logger
}

func NewLoggerEntry() *logrus.Entry {
	return NewLogger().WithFields(logrus.Fields{})
}

// Rand returns random 8 characters
func Rand() string {
	return uuid.New().String()[:8]
}
