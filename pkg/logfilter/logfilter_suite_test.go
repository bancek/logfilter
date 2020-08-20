package logfilter_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var TestCtx context.Context
var TestCtxTimeoutCancel func()
var Logger *logrus.Entry

func TestLogfilter(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(10 * time.Millisecond)

	RunSpecs(t, "Logfilter Suite")
}

var _ = BeforeEach(func() {
	TestCtx, TestCtxTimeoutCancel = context.WithTimeout(context.Background(), 10*time.Second)

	Logger = NewLoggerEntry()
})

var _ = AfterEach(func() {
	TestCtxTimeoutCancel()
})
