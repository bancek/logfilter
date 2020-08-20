package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/bancek/logfilter/pkg/logfilter"
)

func main() {
	baseLogger := logrus.StandardLogger()
	baseLogger.SetFormatter(&logfilter.JSONFormatter{
		JSONFormatter: &logrus.JSONFormatter{
			TimestampFormat: logfilter.RFC3339Milli,
		},
	})

	logger := baseLogger.WithFields(logrus.Fields{
		"service": "logfilter",
	})

	config := logfilter.Config{}

	err := envconfig.Process("logfilter", &config)
	if err != nil {
		logger.Fatal(err)
	}

	logLevel, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	baseLogger.SetLevel(logLevel)

	if len(os.Args) > 1 {
		if len(config.Cmd) > 0 {
			logger.Fatal("Cannot specify both LOGFILTER_CMD and process arguments")
		}

		config.Cmd = os.Args[1:]

		if config.Cmd[0] == "--" {
			config.Cmd = config.Cmd[1:]
		}
	}

	reader := os.Stdin
	writer := os.Stdout

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	go func() {
		select {
		case <-interrupt:
			cancel()
		case <-ctx.Done():
		}
	}()

	logFilter := logfilter.NewLogFilter(&config, reader, writer, logger)

	err = logFilter.Init(ctx)
	if err != nil {
		os.Exit(1)
	}
	defer logFilter.Close()

	err = logFilter.Start()
	if err != nil {
		os.Exit(3)
	}
}
