package logfilter

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

var newLine = []byte{'\n'}

type LogFilter struct {
	config *Config
	reader io.Reader
	writer io.Writer
	logger *logrus.Entry

	ctx      context.Context
	cancel   func()
	errGroup *errgroup.Group

	debugListener net.Listener
	commander     *Commander
	debugServer   *http.Server

	stdoutReader io.ReadCloser
	stdoutWriter io.WriteCloser
	stderrReader io.ReadCloser
	stderrWriter io.WriteCloser

	linesChan chan []byte

	jsonFilter JSONFilter

	fullWriter       io.Writer
	lumberjackLogger *lumberjack.Logger
}

func NewLogFilter(
	config *Config,
	reader io.Reader,
	writer io.Writer,
	logger *logrus.Entry,
) *LogFilter {
	return &LogFilter{
		config: config,
		reader: reader,
		writer: writer,
		logger: logger,
	}
}

func (f *LogFilter) Init(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(ctx)
	f.ctx = ctx
	f.cancel = cancel
	f.errGroup = errGroup

	var err error

	f.debugListener, err = net.Listen("tcp", f.config.DebugListenAddr)
	if err != nil {
		return xerrors.Errorf("debug listener listen failed: %s: %w", f.config.DebugListenAddr, err)
	}

	if len(f.config.Cmd) > 0 {
		f.stdoutReader, f.stdoutWriter = io.Pipe()
		f.stderrReader, f.stderrWriter = io.Pipe()

		f.commander = NewCommander(f.config.Cmd, f.config.CmdShutdownTimeout, f.stdoutWriter, f.stderrWriter, f.logger)
	}

	f.linesChan = make(chan []byte)

	if f.config.ExcludeTemplate != "" && f.config.FilterQuery != "" {
		return xerrors.Errorf("cannot use both exclude template and filter query")
	}

	if f.config.ExcludeTemplate != "" {
		f.logger.WithField("excludeTemplate", f.config.ExcludeTemplate).Debug("Initializing template JSON filter")

		jsonFilter, err := NewTemplateJSONFilter(f.config.ExcludeTemplate)
		if err != nil {
			return xerrors.Errorf("failed to build json filter: %w", err)
		}
		f.jsonFilter = jsonFilter
	} else if f.config.FilterQuery != "" {
		f.logger.WithField("filterQuery", f.config.FilterQuery).Debug("Initializing JQ JSON filter")

		jsonFilter, err := NewJQJSONFilter(f.config.FilterQuery)
		if err != nil {
			return xerrors.Errorf("failed to build json filter: %w", err)
		}
		f.jsonFilter = jsonFilter
	} else {
		f.jsonFilter = StaticJSONFilter(true)
	}

	f.fullWriter = ioutil.Discard

	if f.config.FullOutputFilename != "" {
		f.lumberjackLogger = &lumberjack.Logger{
			Filename:   f.config.FullOutputFilename,
			MaxSize:    f.config.FullOutputMaxSizeMB,
			MaxAge:     f.config.FullOutputMaxAgeDays,
			MaxBackups: f.config.FullOutputMaxBackups,
			Compress:   f.config.FullOutputCompress,
		}

		f.fullWriter = f.lumberjackLogger
	}

	f.debugServer = NewDebugServer()

	return nil
}

func (f *LogFilter) Spawn(fn func(context.Context) error) {
	f.errGroup.Go(func() error {
		err := fn(f.ctx)
		if err != nil {
			f.cancel()
			return err
		}
		return nil
	})
}

func (f *LogFilter) Start() error {
	f.Spawn(func(ctx context.Context) error {
		f.logger.WithField("listenAddr", f.config.DebugListenAddr).Info("Starting debug HTTP server")

		err := f.debugServer.Serve(f.debugListener)
		if ctx.Err() == nil {
			return err
		}
		return nil
	})

	linesDone := make(chan struct{})

	if f.commander == nil {
		scanErrChan := make(chan error, 1)

		// scanning lines from f.reader must not be done in f.Spawn because stdin
		// does not get closed on SIGINT
		go func() {
			err := f.scanLines(f.reader)
			if err == nil {
				err = xerrors.Errorf("reading stdin: %w", io.EOF)
			}
			scanErrChan <- err
		}()

		f.Spawn(func(ctx context.Context) error {
			defer close(linesDone)
			select {
			case err := <-scanErrChan:
				return err
			case <-ctx.Done():
				return nil
			}
		})
	} else {
		var cmdScanningDone sync.WaitGroup

		cmdScanningDone.Add(2)

		f.Spawn(func(_ context.Context) error {
			defer cmdScanningDone.Done()
			return f.scanLines(f.stdoutReader)
		})

		f.Spawn(func(_ context.Context) error {
			defer cmdScanningDone.Done()

			return f.scanLines(f.stderrReader)
		})

		f.Spawn(func(ctx context.Context) error {
			err := f.commander.Start(ctx)
			f.stdoutWriter.Close()
			f.stderrWriter.Close()
			if err == nil {
				err = xerrors.Errorf("command exited")
			}
			return err
		})

		f.Spawn(func(_ context.Context) error {
			cmdScanningDone.Wait()
			close(linesDone)
			return nil
		})
	}

	f.Spawn(func(_ context.Context) error {
		for {
			select {
			case line := <-f.linesChan:
				if f.isLineIncluded(line) {
					if _, err := f.writer.Write(line); err != nil {
						return xerrors.Errorf("writer write failed: %w", err)
					}
					if _, err := f.writer.Write(newLine); err != nil {
						return xerrors.Errorf("writer write failed: %w", err)
					}
				}

				if _, err := f.fullWriter.Write(line); err != nil {
					return xerrors.Errorf("full writer write failed: %w", err)
				}
				if _, err := f.fullWriter.Write(newLine); err != nil {
					return xerrors.Errorf("full writer write failed: %w", err)
				}
			case <-linesDone:
				return nil
			}
		}
	})

	<-f.ctx.Done()

	f.logger.Info("Shutting down")

	_ = f.debugServer.Shutdown(context.Background())

	err := f.errGroup.Wait()
	if err != nil {
		return err
	}

	f.logger.Info("Shutdown")

	return nil
}

func (f *LogFilter) Close() error {
	var closeErr error

	if f.lumberjackLogger != nil {
		if err := f.lumberjackLogger.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close lumberjack logger: %w", err))
		}
	}

	if f.debugListener != nil {
		if err := f.debugListener.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close debug listener: %w", err))
		}
	}

	return closeErr
}

func (f *LogFilter) scanLines(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		b := scanner.Bytes()

		// scanner.Bytes() can return a slice of a bigger byte slice and is not safe to send in channels
		bc := make([]byte, len(b))
		copy(bc, b)

		f.linesChan <- bc
	}

	if err := scanner.Err(); err != nil {
		return xerrors.Errorf("scaning lines error: %w", err)
	}

	return nil
}

func (f *LogFilter) isLineIncluded(line []byte) bool {
	ok, err := f.jsonFilter.IsIncluded(line)
	if err != nil {
		if f.logger.Level <= logrus.DebugLevel {
			f.logger.WithField("line", string(line)).Debug("LogFilter failed to filter line")
		}
		return true
	}
	return ok
}
