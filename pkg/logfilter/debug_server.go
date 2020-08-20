package logfilter

import (
	"net/http"
	_ "net/http/pprof" // register pprof http handlers
)

func NewDebugServer() *http.Server {
	return &http.Server{Handler: nil}
}
