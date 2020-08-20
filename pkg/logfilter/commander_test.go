package logfilter_test

import (
	"context"
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/bancek/logfilter/pkg/logfilter"
)

var _ = Describe("Commander", func() {
	defaultScript := `
		trap 'echo "Caught SIGINT"; exit 0' INT
		while true; do
			sleep 1 &
			wait $!
			echo "Sleep over"
		done
	`

	Describe("Start", func() {
		It("should start the command", func() {
			c := NewCommander([]string{"/bin/sh", "-c", defaultScript}, 500*time.Millisecond, ioutil.Discard, ioutil.Discard, Logger)
			ctx, cancel := context.WithTimeout(TestCtx, 1*time.Second)
			defer cancel()

			Expect(c.Start(ctx)).NotTo(HaveOccurred())
		})

		It("should kill the command if it does not exit after timeout after SIGINT", func() {
			script := `
				trap 'echo "Ignored SIGINT"' INT
				while true; do
					sleep 1 &
					wait $!
					echo "Sleep over"
				done
			`

			c := NewCommander([]string{"/bin/sh", "-c", script}, 1200*time.Millisecond, ioutil.Discard, ioutil.Discard, Logger)
			ctx, cancel := context.WithTimeout(TestCtx, 500*time.Millisecond)
			defer cancel()

			start := time.Now()
			Expect(c.Start(ctx)).To(HaveOccurred())
			Expect(time.Since(start)).To(BeNumerically(">", 600*time.Millisecond))
		})
	})
})
