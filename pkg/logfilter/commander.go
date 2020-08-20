package logfilter

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Commander starts the command.
// It handles graceful shutdown with a timeout after which the command
// is forcefully killed.
type Commander struct {
	cmd                []string
	cmdShutdownTimeout time.Duration
	stdout             io.Writer
	stderr             io.Writer
	logger             *logrus.Entry
}

func NewCommander(
	cmd []string,
	cmdShutdownTimeout time.Duration,
	stdout io.Writer,
	stderr io.Writer,
	logger *logrus.Entry,
) *Commander {
	logger = logger.WithFields(logrus.Fields{
		"cmd": strings.Join(cmd, " "),
	})

	return &Commander{
		cmd:                cmd,
		cmdShutdownTimeout: cmdShutdownTimeout,
		stdout:             stdout,
		stderr:             stderr,
		logger:             logger,
	}
}

func (c *Commander) Start(ctx context.Context) error {
	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmdExited := make(chan struct{}, 1)

	c.logger.Info("Commander starting command")

	cmd := exec.CommandContext(cmdCtx, c.cmd[0], c.cmd[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	process := cmd.Process

	go func() {
		select {
		case <-ctx.Done():
			c.logger.WithFields(logrus.Fields{
				"cmdShutdownTimeout": c.cmdShutdownTimeout,
			}).Info("Commander gracefully shutting down")

			_ = process.Signal(syscall.SIGINT)

			timer := time.NewTimer(c.cmdShutdownTimeout)

			select {
			case <-timer.C:
				c.logger.Warn("Commander forcefully shutting down")
				cancel()
			case <-cmdExited:
				timer.Stop()
			}
		case <-cmdExited:
		}
	}()

	err := cmd.Wait()

	cmdExited <- struct{}{}

	if err == nil {
		c.logger.Info("Commander command successfully exited")
	} else {
		c.logger.WithError(err).Info("Commander command exited with error")
	}

	return err
}
