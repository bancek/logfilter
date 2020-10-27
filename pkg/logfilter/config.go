package logfilter

import (
	"time"

	"github.com/kballard/go-shellquote"
)

// Config contains the configuration for the logfilter
type Config struct {
	// Cmd is the command that logfilter will run and use as the input. If empty
	// process args will be used as the command. If that is empty stdin will be
	// used as the input. Arguments are separated using spaces and can be quoted.
	// (LOGFILTER_CMD)
	Cmd Cmd

	// CmdShutdownTimeout is the timeout after which the command will be
	// forecefully killed after the logfilter is stopped. Command will first
	// receive SIGINT and then SIGKILL after CmdShutdownTimeout.
	// (LOGFILTER_CMDSHUTDOWNTIMEOUT)
	CmdShutdownTimeout time.Duration `default:"10s"`

	// ExcludeTemplate is a Go text/template. If it renders a value "true" the
	// following JSON will be excluded from the stdout. The template can render
	// multiple "true" values to simplify the exclusion logic.
	// (LOGFILTER_EXCLUDETEMPLATE)
	ExcludeTemplate string

	// FilterQuery is a JQ query. You can use `select(.k1 != "v1") | select(.k2 !=
	// "v2")` to filter the JSON lines.
	// (LOGFILTER_FILTER_QUERY)
	FilterQuery string

	// DebugListenAddr is the address of the HTTP debug (pprof) server
	// (LOGFILTER_DEBUGLISTENADDR).
	DebugListenAddr string `default:"localhost:4083"`

	// FullOutputFilename is file to write the full logs to. Backup log files will
	// be retained in the same directory. If empty the full output will be
	// discarded.
	// (LOGFILTER_FULLOUTPUTFILENAME)
	FullOutputFilename string

	// FullOutputMaxSizeMB is the maximum size in megabytes of the log file
	// before it gets rotated. It defaults to 100 megabytes.
	// (LOGFILTER_FULLOUTPUTMAXSIZEMB)
	FullOutputMaxSizeMB int `default:"100"`

	// FullOutputMaxAgeDays is the maximum number of days to retain old log
	// files based on the timestamp encoded in their filename.  Note that a day is
	// defined as 24 hours and may not exactly correspond to calendar days due to
	// daylight savings, leap seconds, etc. The default is not to remove old log
	// files based on age.
	// (LOGFILTER_FULLOUTPUTMAXAGEDAYS)
	FullOutputMaxAgeDays int

	// FullOutputMaxBackups is the maximum number of old log files to retain.
	// The default is to retain all old log files (though FullOutputMaxAgeDays
	// may still cause them to get deleted.)
	// (LOGFILTER_FULLOUTPUTMAXBACKUPS)
	FullOutputMaxBackups int

	// FullOutputCompress determines if the rotated log files should be
	// compressed using gzip. The default is not to perform compression.
	// (LOGFILTER_FULLOUTPUTCOMPRESS)
	FullOutputCompress bool

	// MaxScanLineSize is the maximum size used to buffer lines.
	// (LOGFILTER_MAXSCANLINESIZE)
	MaxScanLineSize int `default:"52428800"`

	// LogLevel is the log level of the logfilter.
	// (LOGFILTER_LOGLEVEL)
	LogLevel string `default:"info"`
}

type Cmd []string

func (c *Cmd) Decode(value string) error {
	cmd, err := shellquote.Split(value)
	if err != nil {
		return err
	}
	*c = cmd
	return nil
}
