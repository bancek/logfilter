package logfilter

import (
	"time"

	"github.com/sirupsen/logrus"
)

const RFC3339Milli = "2006-01-02T15:04:05.999Z07:00"

type JSONFormatter struct {
	*logrus.JSONFormatter
}

func (f *JSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	for k, v := range entry.Data {
		switch v := v.(type) {
		case time.Duration:
			entry.Data[k] = float32(v) / 1000000
		case time.Time:
			entry.Data[k] = v.Format(RFC3339Milli)
		}
	}

	return f.JSONFormatter.Format(entry)
}
